package payment_test

import (
	"context"
	"errors"
	"testing"

	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/valueobject"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	ctx         = context.Background()
	errDatabase = errors.New("database unavailable")
)

func pendingBudget(t *testing.T) *entity.Budget {
	t.Helper()
	item, err := entity.NewBudgetItem(valueobject.BudgetItemKindService, "service-1", "Troca de óleo", 1, 120)
	require.NoError(t, err)
	created, err := entity.NewBudget("order-1", "saga-1", entity.BudgetCustomer{ID: "customer-1", Name: "Maria Silva", Email: "maria@example.com"}, []*entity.BudgetItem{item})
	require.NoError(t, err)
	return created
}

func approvedBudget(t *testing.T) *entity.Budget {
	t.Helper()
	created := pendingBudget(t)
	require.NoError(t, created.Approve())
	return created
}

func pendingPayment(t *testing.T) *entity.Payment {
	t.Helper()
	created, err := entity.NewPayment(approvedBudget(t), "fake")
	require.NoError(t, err)
	require.NoError(t, created.AttachCheckout("pref-1", "https://checkout.example.com/pref-1"))
	return created
}

func confirmedPayment(t *testing.T) *entity.Payment {
	t.Helper()
	created := pendingPayment(t)
	require.NoError(t, created.Confirm("mp-123"))
	return created
}

func replyOfType(messageType string) any {
	return mock.MatchedBy(func(outgoing []messaging.OutgoingMessage) bool {
		return len(outgoing) == 1 &&
			outgoing[0].Destination == messaging.DestinationSagaReplies &&
			outgoing[0].Message.Type == messageType &&
			outgoing[0].Message.SagaID == "saga-1" &&
			outgoing[0].Message.OrderID == "order-1"
	})
}
