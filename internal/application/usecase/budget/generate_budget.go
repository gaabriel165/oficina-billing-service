package budget

import (
	"context"
	"errors"
	"log/slog"

	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/valueobject"
)

const singleUnit = 1

type GenerateBudgetInput struct {
	OrderID  string
	SagaID   string
	Customer entity.BudgetCustomer
	Services []messaging.ServiceLine
	Parts    []messaging.PartLine
}

type GenerateBudgetUseCase struct {
	budgetRepo    repository.BudgetRepository
	notifications repository.NotificationService
}

func NewGenerateBudgetUseCase(
	budgetRepo repository.BudgetRepository,
	notifications repository.NotificationService,
) *GenerateBudgetUseCase {
	return &GenerateBudgetUseCase{budgetRepo: budgetRepo, notifications: notifications}
}

func (u *GenerateBudgetUseCase) Execute(ctx context.Context, input GenerateBudgetInput) (*entity.Budget, error) {
	existing, err := u.budgetRepo.FindByOrderID(ctx, input.OrderID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, repository.ErrBudgetNotFound) {
		return nil, err
	}

	items, err := itemsFromLines(input.Services, input.Parts)
	if err != nil {
		return nil, err
	}

	budget, err := entity.NewBudget(input.OrderID, input.SagaID, input.Customer, items)
	if err != nil {
		return nil, err
	}

	generated, err := usecase.Reply(messaging.TypeBudgetGenerated, budget.SagaID(), budget.OrderID(), messaging.BudgetGeneratedPayload{
		BudgetID:    budget.ID(),
		TotalAmount: budget.TotalAmount(),
	})
	if err != nil {
		return nil, err
	}

	if err := u.budgetRepo.Create(ctx, budget, generated); err != nil {
		return nil, err
	}

	u.notify(ctx, budget)
	return budget, nil
}

func (u *GenerateBudgetUseCase) notify(ctx context.Context, budget *entity.Budget) {
	if err := u.notifications.NotifyBudgetReady(ctx, budget); err != nil {
		slog.ErrorContext(ctx, "integration.error",
			slog.String("event", "integration.error"),
			slog.String("integration", "resend"),
			slog.String("operation", "notify_budget_ready"),
			slog.String("order_id", budget.OrderID()),
			slog.String("error", err.Error()),
		)
	}
}

func itemsFromLines(services []messaging.ServiceLine, parts []messaging.PartLine) ([]*entity.BudgetItem, error) {
	items := make([]*entity.BudgetItem, 0, len(services)+len(parts))
	for _, service := range services {
		item, err := entity.NewBudgetItem(valueobject.BudgetItemKindService, service.ServiceID, service.Name, singleUnit, service.LaborPrice)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	for _, part := range parts {
		item, err := entity.NewBudgetItem(valueobject.BudgetItemKindPart, part.PartID, part.Name, part.Quantity, part.UnitPrice)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}
