package budget

import (
	"context"

	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
)

type GetBudgetUseCase struct {
	budgetRepo repository.BudgetRepository
}

func NewGetBudgetUseCase(budgetRepo repository.BudgetRepository) *GetBudgetUseCase {
	return &GetBudgetUseCase{budgetRepo: budgetRepo}
}

func (u *GetBudgetUseCase) Execute(ctx context.Context, budgetID string, requester Requester) (*entity.Budget, error) {
	return findAccessible(ctx, u.budgetRepo, budgetID, requester)
}

type ListBudgetsUseCase struct {
	budgetRepo repository.BudgetRepository
}

func NewListBudgetsUseCase(budgetRepo repository.BudgetRepository) *ListBudgetsUseCase {
	return &ListBudgetsUseCase{budgetRepo: budgetRepo}
}

func (u *ListBudgetsUseCase) Execute(ctx context.Context, orderID string, requester Requester) ([]*entity.Budget, error) {
	if orderID != "" {
		return u.findByOrder(ctx, orderID, requester)
	}
	if requester.IsCustomer {
		return u.budgetRepo.FindByCustomerID(ctx, requester.ID)
	}
	return u.budgetRepo.FindAll(ctx)
}

func (u *ListBudgetsUseCase) findByOrder(ctx context.Context, orderID string, requester Requester) ([]*entity.Budget, error) {
	budget, err := u.budgetRepo.FindByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if !requester.canAccess(budget) {
		return []*entity.Budget{}, nil
	}
	return []*entity.Budget{budget}, nil
}
