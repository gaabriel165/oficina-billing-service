package budget

import (
	"context"
	"errors"

	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
)

type CancelBudgetInput struct {
	OrderID string
	SagaID  string
}

type CancelBudgetUseCase struct {
	budgetRepo repository.BudgetRepository
	outbox     messaging.OutboxRepository
}

func NewCancelBudgetUseCase(budgetRepo repository.BudgetRepository, outbox messaging.OutboxRepository) *CancelBudgetUseCase {
	return &CancelBudgetUseCase{budgetRepo: budgetRepo, outbox: outbox}
}

func (u *CancelBudgetUseCase) Execute(ctx context.Context, input CancelBudgetInput) error {
	cancelled, err := usecase.Reply(messaging.TypeBudgetCancelled, input.SagaID, input.OrderID, nil)
	if err != nil {
		return err
	}

	budget, err := u.budgetRepo.FindByOrderID(ctx, input.OrderID)
	if errors.Is(err, repository.ErrBudgetNotFound) {
		return u.outbox.Enqueue(ctx, cancelled)
	}
	if err != nil {
		return err
	}

	if budget.IsCancelled() {
		return u.outbox.Enqueue(ctx, cancelled)
	}

	if err := budget.Cancel(); err != nil {
		return err
	}
	return u.budgetRepo.Update(ctx, budget, cancelled)
}
