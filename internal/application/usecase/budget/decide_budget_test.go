package budget_test

import (
	"testing"

	"github.com/gabrielcamargo/oficina-billing-service/internal/application/mocks"
	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase"
	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase/budget"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/valueobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApproveBudget_ShouldApproveAndReplyToSaga(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	existing := pendingBudget(t)
	budgetRepo.On("FindByID", ctx, existing.ID()).Return(existing, nil)
	budgetRepo.On("Update", ctx, existing, replyOfType(messaging.TypeBudgetApproved)).Return(nil)

	result, err := budget.NewApproveBudgetUseCase(budgetRepo).Execute(ctx, existing.ID(), owner)

	require.NoError(t, err)
	assert.True(t, result.IsApproved())
}

func TestApproveBudget_ShouldAllowOperator(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	existing := pendingBudget(t)
	budgetRepo.On("FindByID", ctx, existing.ID()).Return(existing, nil)
	budgetRepo.On("Update", ctx, existing, replyOfType(messaging.TypeBudgetApproved)).Return(nil)

	result, err := budget.NewApproveBudgetUseCase(budgetRepo).Execute(ctx, existing.ID(), operator)

	require.NoError(t, err)
	assert.True(t, result.IsApproved())
}

func TestApproveBudget_ShouldForbidAnotherCustomer(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	existing := pendingBudget(t)
	budgetRepo.On("FindByID", ctx, existing.ID()).Return(existing, nil)

	_, err := budget.NewApproveBudgetUseCase(budgetRepo).Execute(ctx, existing.ID(), otherCustomer)

	assert.ErrorIs(t, err, usecase.ErrForbidden)
	assert.False(t, existing.IsApproved())
}

func TestApproveBudget_ShouldReturnErrorWhenNotFound(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	budgetRepo.On("FindByID", ctx, "missing").Return(nil, repository.ErrBudgetNotFound)

	_, err := budget.NewApproveBudgetUseCase(budgetRepo).Execute(ctx, "missing", operator)

	assert.ErrorIs(t, err, repository.ErrBudgetNotFound)
}

func TestApproveBudget_ShouldReturnErrorWhenAlreadyDecided(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	existing := pendingBudget(t)
	require.NoError(t, existing.Reject("too expensive"))
	budgetRepo.On("FindByID", ctx, existing.ID()).Return(existing, nil)

	_, err := budget.NewApproveBudgetUseCase(budgetRepo).Execute(ctx, existing.ID(), owner)

	assert.ErrorIs(t, err, entity.ErrBudgetInvalidTransition)
}

func TestApproveBudget_ShouldReturnErrorWhenSaveFails(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	existing := pendingBudget(t)
	budgetRepo.On("FindByID", ctx, existing.ID()).Return(existing, nil)
	budgetRepo.On("Update", ctx, existing, replyOfType(messaging.TypeBudgetApproved)).Return(errDatabase)

	_, err := budget.NewApproveBudgetUseCase(budgetRepo).Execute(ctx, existing.ID(), owner)

	assert.ErrorIs(t, err, errDatabase)
}

func TestRejectBudget_ShouldRejectAndReplyToSaga(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	existing := pendingBudget(t)
	budgetRepo.On("FindByID", ctx, existing.ID()).Return(existing, nil)
	budgetRepo.On("Update", ctx, existing, replyOfType(messaging.TypeBudgetRejected)).Return(nil)

	result, err := budget.NewRejectBudgetUseCase(budgetRepo).Execute(ctx, budget.RejectBudgetInput{
		BudgetID: existing.ID(), Reason: "too expensive", Requester: owner,
	})

	require.NoError(t, err)
	assert.Equal(t, valueobject.BudgetStatusRejected, result.Status())
	assert.Equal(t, "too expensive", result.RejectionReason())
}

func TestRejectBudget_ShouldRequireReason(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	existing := pendingBudget(t)
	budgetRepo.On("FindByID", ctx, existing.ID()).Return(existing, nil)

	_, err := budget.NewRejectBudgetUseCase(budgetRepo).Execute(ctx, budget.RejectBudgetInput{BudgetID: existing.ID(), Requester: owner})

	assert.ErrorIs(t, err, entity.ErrBudgetReasonRequired)
}

func TestRejectBudget_ShouldForbidAnotherCustomer(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	existing := pendingBudget(t)
	budgetRepo.On("FindByID", ctx, existing.ID()).Return(existing, nil)

	_, err := budget.NewRejectBudgetUseCase(budgetRepo).Execute(ctx, budget.RejectBudgetInput{
		BudgetID: existing.ID(), Reason: "too expensive", Requester: otherCustomer,
	})

	assert.ErrorIs(t, err, usecase.ErrForbidden)
}

func TestRejectBudget_ShouldReturnErrorWhenSaveFails(t *testing.T) {
	budgetRepo := mocks.NewMockBudgetRepository(t)
	existing := pendingBudget(t)
	budgetRepo.On("FindByID", ctx, existing.ID()).Return(existing, nil)
	budgetRepo.On("Update", ctx, existing, replyOfType(messaging.TypeBudgetRejected)).Return(errDatabase)

	_, err := budget.NewRejectBudgetUseCase(budgetRepo).Execute(ctx, budget.RejectBudgetInput{
		BudgetID: existing.ID(), Reason: "too expensive", Requester: owner,
	})

	assert.ErrorIs(t, err, errDatabase)
}
