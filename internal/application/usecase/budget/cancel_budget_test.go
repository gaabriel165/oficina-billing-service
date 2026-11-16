package budget_test

import (
	"testing"

	"github.com/gabrielcamargo/oficina-billing-service/internal/application/mocks"
	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase/budget"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var cancelInput = budget.CancelBudgetInput{OrderID: "order-1", SagaID: "saga-1"}

func TestCancelBudget_ShouldCancelAndReplyToSaga(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	outbox := mocks.NewMockOutboxRepository(t)
	existing := pendingBudget(t)
	budgetRepo.On("FindByOrderID", ctx, "order-1").Return(existing, nil)
	budgetRepo.On("Update", ctx, existing, replyOfType(messaging.TypeBudgetCancelled)).Return(nil)

	err := budget.NewCancelBudgetUseCase(budgetRepo, outbox).Execute(ctx, cancelInput)

	require.NoError(t, err)
	assert.True(t, existing.IsCancelled())
}

func TestCancelBudget_ShouldOnlyReplyWhenBudgetDoesNotExist(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	outbox := mocks.NewMockOutboxRepository(t)
	budgetRepo.On("FindByOrderID", ctx, "order-1").Return(nil, repository.ErrBudgetNotFound)
	outbox.On("Enqueue", ctx, replyOfType(messaging.TypeBudgetCancelled)).Return(nil)

	err := budget.NewCancelBudgetUseCase(budgetRepo, outbox).Execute(ctx, cancelInput)

	assert.NoError(t, err)
}

func TestCancelBudget_ShouldOnlyReplyWhenAlreadyCancelled(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	outbox := mocks.NewMockOutboxRepository(t)
	existing := pendingBudget(t)
	require.NoError(t, existing.Cancel())
	budgetRepo.On("FindByOrderID", ctx, "order-1").Return(existing, nil)
	outbox.On("Enqueue", ctx, replyOfType(messaging.TypeBudgetCancelled)).Return(nil)

	err := budget.NewCancelBudgetUseCase(budgetRepo, outbox).Execute(ctx, cancelInput)

	assert.NoError(t, err)
}

func TestCancelBudget_ShouldReturnErrorWhenLookupFails(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	outbox := mocks.NewMockOutboxRepository(t)
	budgetRepo.On("FindByOrderID", ctx, "order-1").Return(nil, errDatabase)

	err := budget.NewCancelBudgetUseCase(budgetRepo, outbox).Execute(ctx, cancelInput)

	assert.ErrorIs(t, err, errDatabase)
}

func TestCancelBudget_ShouldReturnErrorWhenSagaIsMissing(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	outbox := mocks.NewMockOutboxRepository(t)

	err := budget.NewCancelBudgetUseCase(budgetRepo, outbox).Execute(ctx, budget.CancelBudgetInput{OrderID: "order-1"})

	assert.ErrorIs(t, err, messaging.ErrMessageSagaRequired)
}
