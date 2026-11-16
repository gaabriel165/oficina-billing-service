package budget

import (
	"context"

	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
)

type Requester struct {
	ID         string
	IsCustomer bool
}

func (r Requester) canAccess(budget *entity.Budget) bool {
	return !r.IsCustomer || budget.BelongsTo(r.ID)
}

type ApproveBudgetUseCase struct {
	budgetRepo repository.BudgetRepository
}

func NewApproveBudgetUseCase(budgetRepo repository.BudgetRepository) *ApproveBudgetUseCase {
	return &ApproveBudgetUseCase{budgetRepo: budgetRepo}
}

func (u *ApproveBudgetUseCase) Execute(ctx context.Context, budgetID string, requester Requester) (*entity.Budget, error) {
	budget, err := findAccessible(ctx, u.budgetRepo, budgetID, requester)
	if err != nil {
		return nil, err
	}

	if err := budget.Approve(); err != nil {
		return nil, err
	}

	approved, err := usecase.Reply(messaging.TypeBudgetApproved, budget.SagaID(), budget.OrderID(), messaging.BudgetDecisionPayload{
		BudgetID: budget.ID(),
	})
	if err != nil {
		return nil, err
	}

	if err := u.budgetRepo.Update(ctx, budget, approved); err != nil {
		return nil, err
	}
	return budget, nil
}

type RejectBudgetInput struct {
	BudgetID  string
	Reason    string
	Requester Requester
}

type RejectBudgetUseCase struct {
	budgetRepo repository.BudgetRepository
}

func NewRejectBudgetUseCase(budgetRepo repository.BudgetRepository) *RejectBudgetUseCase {
	return &RejectBudgetUseCase{budgetRepo: budgetRepo}
}

func (u *RejectBudgetUseCase) Execute(ctx context.Context, input RejectBudgetInput) (*entity.Budget, error) {
	budget, err := findAccessible(ctx, u.budgetRepo, input.BudgetID, input.Requester)
	if err != nil {
		return nil, err
	}

	if err := budget.Reject(input.Reason); err != nil {
		return nil, err
	}

	rejected, err := usecase.Reply(messaging.TypeBudgetRejected, budget.SagaID(), budget.OrderID(), messaging.BudgetDecisionPayload{
		BudgetID: budget.ID(),
		Reason:   input.Reason,
	})
	if err != nil {
		return nil, err
	}

	if err := u.budgetRepo.Update(ctx, budget, rejected); err != nil {
		return nil, err
	}
	return budget, nil
}

func findAccessible(ctx context.Context, budgetRepo repository.BudgetRepository, budgetID string, requester Requester) (*entity.Budget, error) {
	budget, err := budgetRepo.FindByID(ctx, budgetID)
	if err != nil {
		return nil, err
	}
	if !requester.canAccess(budget) {
		return nil, usecase.ErrForbidden
	}
	return budget, nil
}
