package api_test

import (
	"context"
	"testing"

	"github.com/gabrielcamargo/oficina-billing-service/configs"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type handlerRegistry map[string]messaging.Handler

func (r handlerRegistry) Register(messageType string, handler messaging.Handler) {
	r[messageType] = handler
}

func registeredHandlers(environment *testEnvironment) handlerRegistry {
	registry := handlerRegistry{}
	environment.server.RegisterMessageHandlers(registry)
	return registry
}

func command(t *testing.T, messageType string, payload any) messaging.Message {
	t.Helper()
	message, err := messaging.NewMessage(messageType, "saga-1", "order-1", payload)
	require.NoError(t, err)
	return message
}

func TestMessageHandlers_ShouldRegisterEveryBillingCommand(t *testing.T) {
	registry := registeredHandlers(newTestEnvironment(t, configs.PaymentGatewayFake))

	assert.Len(t, registry, 4)
	assert.Contains(t, registry, messaging.TypeGenerateBudget)
	assert.Contains(t, registry, messaging.TypeCancelBudget)
	assert.Contains(t, registry, messaging.TypeRequestPayment)
	assert.Contains(t, registry, messaging.TypeRefundPayment)
}

func TestGenerateBudgetCommand_ShouldCreateBudgetFromDiagnosis(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)
	environment.budgetRepo.On("FindByOrderID", mock.Anything, "order-1").Return(nil, repository.ErrBudgetNotFound)
	environment.budgetRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Budget"), replyOfType(messaging.TypeBudgetGenerated)).Return(nil)
	environment.notifications.On("NotifyBudgetReady", mock.Anything, mock.AnythingOfType("*entity.Budget")).Return(nil)
	message := command(t, messaging.TypeGenerateBudget, messaging.GenerateBudgetPayload{
		CustomerID:   "customer-1",
		CustomerName: "Maria Silva",
		Services:     []messaging.ServiceLine{{ServiceID: "service-1", Name: "Troca de óleo", LaborPrice: 120}},
	})

	err := registeredHandlers(environment)[messaging.TypeGenerateBudget](context.Background(), message)

	assert.NoError(t, err)
}

func TestGenerateBudgetCommand_ShouldRejectInvalidPayload(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)
	message := command(t, messaging.TypeGenerateBudget, nil)
	message.Payload = []byte(`{"services":"invalid"}`)

	err := registeredHandlers(environment)[messaging.TypeGenerateBudget](context.Background(), message)

	assert.ErrorIs(t, err, messaging.ErrMessagePayloadInvalid)
}

func TestCancelBudgetCommand_ShouldCancelBudget(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)
	budget := pendingBudget(t)
	environment.budgetRepo.On("FindByOrderID", mock.Anything, "order-1").Return(budget, nil)
	environment.budgetRepo.On("Update", mock.Anything, budget, replyOfType(messaging.TypeBudgetCancelled)).Return(nil)

	err := registeredHandlers(environment)[messaging.TypeCancelBudget](context.Background(), command(t, messaging.TypeCancelBudget, nil))

	require.NoError(t, err)
	assert.True(t, budget.IsCancelled())
}

func TestRequestPaymentCommand_ShouldKeepExistingPayment(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)
	environment.paymentRepo.On("FindByOrderID", mock.Anything, "order-1").Return(pendingPayment(t), nil)

	err := registeredHandlers(environment)[messaging.TypeRequestPayment](context.Background(), command(t, messaging.TypeRequestPayment, nil))

	assert.NoError(t, err)
}

func TestRefundPaymentCommand_ShouldAcknowledgeWhenThereIsNoPayment(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)
	environment.paymentRepo.On("FindByOrderID", mock.Anything, "order-1").Return(nil, repository.ErrPaymentNotFound)
	environment.outbox.On("Enqueue", mock.Anything, replyOfType(messaging.TypePaymentRefunded)).Return(nil)

	err := registeredHandlers(environment)[messaging.TypeRefundPayment](context.Background(), command(t, messaging.TypeRefundPayment, nil))

	assert.NoError(t, err)
}
