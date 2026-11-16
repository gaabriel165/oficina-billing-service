package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/gabrielcamargo/oficina-billing-service/internal/application/outbox"
	domain "github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-billing-service/internal/infrastructure/messaging"
	infrarepo "github.com/gabrielcamargo/oficina-billing-service/internal/infrastructure/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testCommandsQueue = "oficina-billing-commands"
	testRepliesQueue  = "oficina-saga-replies"
)

type collectingHandler struct {
	received chan domain.Message
}

func (h *collectingHandler) handle(_ context.Context, message domain.Message) error {
	h.received <- message
	return nil
}

func TestProcessedMessages_ShouldRememberSavedMessage(t *testing.T) {
	processed := infrarepo.NewGormProcessedMessageRepository(newTestDB(t))
	messageID := uuid.NewString()
	ctx := context.Background()

	existsBefore, err := processed.Exists(ctx, messageID)
	require.NoError(t, err)
	require.NoError(t, processed.Save(ctx, messageID))
	require.NoError(t, processed.Save(ctx, messageID))
	existsAfter, err := processed.Exists(ctx, messageID)
	require.NoError(t, err)

	assert.False(t, existsBefore)
	assert.True(t, existsAfter)
}

func TestConsumer_ShouldReceiveCommandPublishedToQueue(t *testing.T) {
	client := newTestQueueClient(t, testCommandsQueue)
	db := newTestDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	publisher := messaging.NewSQSPublisher(client, map[domain.Destination]string{
		domain.DestinationBillingCommands: testCommandsQueue,
	})
	command, err := domain.NewMessage(domain.TypeCancelBudget, uuid.NewString(), uuid.NewString(), domain.ReasonPayload{Reason: "payment failed"})
	require.NoError(t, err)
	require.NoError(t, publisher.Publish(ctx, domain.DestinationBillingCommands, command))

	handler := &collectingHandler{received: make(chan domain.Message, 1)}
	consumer := messaging.NewSQSConsumer(client, testCommandsQueue, handler.handle, infrarepo.NewGormProcessedMessageRepository(db))
	go consumer.Run(ctx)

	select {
	case received := <-handler.received:
		var payload domain.ReasonPayload
		require.NoError(t, received.DecodePayload(&payload))
		assert.Equal(t, command.ID, received.ID)
		assert.Equal(t, "payment failed", payload.Reason)
	case <-time.After(20 * time.Second):
		t.Fatal("command was not consumed")
	}
}

func TestOutboxRelay_ShouldDoNothingWhenOutboxIsEmpty(t *testing.T) {
	client := newTestQueueClient(t, testRepliesQueue)
	db := newTestDB(t)
	publisher := messaging.NewSQSPublisher(client, map[domain.Destination]string{
		domain.DestinationSagaReplies: testRepliesQueue,
	})

	relayed, err := outbox.NewRelay(infrarepo.NewGormOutboxRepository(db), publisher).RelayOnce(context.Background())

	require.NoError(t, err)
	assert.Zero(t, relayed)
}
