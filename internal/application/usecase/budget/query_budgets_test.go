package budget_test

import (
	"testing"

	"github.com/gabrielcamargo/oficina-billing-service/internal/application/mocks"
	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase"
	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase/budget"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetBudget_ShouldReturnBudgetToOwner(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	existing := pendingBudget(t)
	budgetRepo.On("FindByID", ctx, existing.ID()).Return(existing, nil)

	result, err := budget.NewGetBudgetUseCase(budgetRepo).Execute(ctx, existing.ID(), owner)

	require.NoError(t, err)
	assert.Equal(t, existing.ID(), result.ID())
}

func TestGetBudget_ShouldForbidAnotherCustomer(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	existing := pendingBudget(t)
	budgetRepo.On("FindByID", ctx, existing.ID()).Return(existing, nil)

	_, err := budget.NewGetBudgetUseCase(budgetRepo).Execute(ctx, existing.ID(), otherCustomer)

	assert.ErrorIs(t, err, usecase.ErrForbidden)
}

func TestListBudgets_ShouldReturnAllToOperator(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	budgetRepo.On("FindAll", ctx).Return([]*entity.Budget{pendingBudget(t), pendingBudget(t)}, nil)

	result, err := budget.NewListBudgetsUseCase(budgetRepo).Execute(ctx, "", operator)

	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestListBudgets_ShouldReturnOnlyOwnBudgetsToCustomer(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	budgetRepo.On("FindByCustomerID", ctx, "customer-1").Return([]*entity.Budget{pendingBudget(t)}, nil)

	result, err := budget.NewListBudgetsUseCase(budgetRepo).Execute(ctx, "", owner)

	require.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestListBudgets_ShouldFilterByOrder(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	existing := pendingBudget(t)
	budgetRepo.On("FindByOrderID", ctx, "order-1").Return(existing, nil)

	result, err := budget.NewListBudgetsUseCase(budgetRepo).Execute(ctx, "order-1", owner)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, existing.ID(), result[0].ID())
}

func TestListBudgets_ShouldHideOrderBudgetFromAnotherCustomer(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	budgetRepo.On("FindByOrderID", ctx, "order-1").Return(pendingBudget(t), nil)

	result, err := budget.NewListBudgetsUseCase(budgetRepo).Execute(ctx, "order-1", otherCustomer)

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestListBudgets_ShouldReturnErrorWhenOrderHasNoBudget(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	budgetRepo.On("FindByOrderID", ctx, "order-9").Return(nil, repository.ErrBudgetNotFound)

	_, err := budget.NewListBudgetsUseCase(budgetRepo).Execute(ctx, "order-9", operator)

	assert.ErrorIs(t, err, repository.ErrBudgetNotFound)
}
