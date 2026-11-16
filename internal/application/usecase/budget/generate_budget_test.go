package budget_test

import (
	"errors"
	"testing"

	"github.com/gabrielcamargo/oficina-billing-service/internal/application/mocks"
	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase/budget"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/valueobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func generateInput() budget.GenerateBudgetInput {
	return budget.GenerateBudgetInput{
		OrderID:  "order-1",
		SagaID:   "saga-1",
		Customer: entity.BudgetCustomer{ID: "customer-1", Name: "Maria Silva", Email: "maria@example.com"},
		Services: []messaging.ServiceLine{{ServiceID: "service-1", Name: "Troca de óleo", LaborPrice: 120}},
		Parts:    []messaging.PartLine{{PartID: "part-1", Name: "Filtro de óleo", Quantity: 2, UnitPrice: 30}},
	}
}

func TestGenerateBudget_ShouldCreateBudgetAndReplyToSaga(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	notifications := mocks.NewMockNotificationService(t)
	budgetRepo.On("FindByOrderID", ctx, "order-1").Return(nil, repository.ErrBudgetNotFound)
	budgetRepo.On("Create", ctx, mock.AnythingOfType("*entity.Budget"), replyOfType(messaging.TypeBudgetGenerated)).Return(nil)
	notifications.On("NotifyBudgetReady", ctx, mock.AnythingOfType("*entity.Budget")).Return(nil)

	result, err := budget.NewGenerateBudgetUseCase(budgetRepo, notifications).Execute(ctx, generateInput())

	require.NoError(t, err)
	assert.Equal(t, valueobject.BudgetStatusPendingApproval, result.Status())
	assert.Equal(t, 180.0, result.TotalAmount())
	assert.Len(t, result.Items(), 2)
}

func TestGenerateBudget_ShouldReturnExistingBudgetWhenCommandIsRepeated(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	notifications := mocks.NewMockNotificationService(t)
	existing := pendingBudget(t)
	budgetRepo.On("FindByOrderID", ctx, "order-1").Return(existing, nil)

	result, err := budget.NewGenerateBudgetUseCase(budgetRepo, notifications).Execute(ctx, generateInput())

	require.NoError(t, err)
	assert.Equal(t, existing.ID(), result.ID())
}

func TestGenerateBudget_ShouldSucceedWhenNotificationFails(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	notifications := mocks.NewMockNotificationService(t)
	budgetRepo.On("FindByOrderID", ctx, "order-1").Return(nil, repository.ErrBudgetNotFound)
	budgetRepo.On("Create", ctx, mock.AnythingOfType("*entity.Budget"), replyOfType(messaging.TypeBudgetGenerated)).Return(nil)
	notifications.On("NotifyBudgetReady", ctx, mock.AnythingOfType("*entity.Budget")).Return(errors.New("resend unavailable"))

	result, err := budget.NewGenerateBudgetUseCase(budgetRepo, notifications).Execute(ctx, generateInput())

	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGenerateBudget_ShouldReturnErrorWhenLookupFails(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	notifications := mocks.NewMockNotificationService(t)
	budgetRepo.On("FindByOrderID", ctx, "order-1").Return(nil, errDatabase)

	_, err := budget.NewGenerateBudgetUseCase(budgetRepo, notifications).Execute(ctx, generateInput())

	assert.ErrorIs(t, err, errDatabase)
}

func TestGenerateBudget_ShouldReturnErrorWhenServiceLineIsInvalid(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	notifications := mocks.NewMockNotificationService(t)
	budgetRepo.On("FindByOrderID", ctx, "order-1").Return(nil, repository.ErrBudgetNotFound)
	input := generateInput()
	input.Services[0].LaborPrice = 0

	_, err := budget.NewGenerateBudgetUseCase(budgetRepo, notifications).Execute(ctx, input)

	assert.ErrorIs(t, err, entity.ErrBudgetItemInvalidPrice)
}

func TestGenerateBudget_ShouldReturnErrorWhenPartLineIsInvalid(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	notifications := mocks.NewMockNotificationService(t)
	budgetRepo.On("FindByOrderID", ctx, "order-1").Return(nil, repository.ErrBudgetNotFound)
	input := generateInput()
	input.Parts[0].Quantity = 0

	_, err := budget.NewGenerateBudgetUseCase(budgetRepo, notifications).Execute(ctx, input)

	assert.ErrorIs(t, err, entity.ErrBudgetItemInvalidQty)
}

func TestGenerateBudget_ShouldReturnErrorWhenDiagnosisHasNoItems(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	notifications := mocks.NewMockNotificationService(t)
	budgetRepo.On("FindByOrderID", ctx, "order-1").Return(nil, repository.ErrBudgetNotFound)
	input := generateInput()
	input.Services = nil
	input.Parts = nil

	_, err := budget.NewGenerateBudgetUseCase(budgetRepo, notifications).Execute(ctx, input)

	assert.ErrorIs(t, err, entity.ErrBudgetNoItems)
}

func TestGenerateBudget_ShouldReturnErrorWhenSagaIsMissing(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	notifications := mocks.NewMockNotificationService(t)
	budgetRepo.On("FindByOrderID", ctx, "order-1").Return(nil, repository.ErrBudgetNotFound)
	input := generateInput()
	input.SagaID = ""

	_, err := budget.NewGenerateBudgetUseCase(budgetRepo, notifications).Execute(ctx, input)

	assert.ErrorIs(t, err, entity.ErrBudgetSagaRequired)
}

func TestGenerateBudget_ShouldReturnErrorWhenSaveFails(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	notifications := mocks.NewMockNotificationService(t)
	budgetRepo.On("FindByOrderID", ctx, "order-1").Return(nil, repository.ErrBudgetNotFound)
	budgetRepo.On("Create", ctx, mock.AnythingOfType("*entity.Budget"), replyOfType(messaging.TypeBudgetGenerated)).Return(errDatabase)

	_, err := budget.NewGenerateBudgetUseCase(budgetRepo, notifications).Execute(ctx, generateInput())

	assert.ErrorIs(t, err, errDatabase)
}
