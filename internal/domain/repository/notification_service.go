package repository

import (
	"context"

	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
)

type NotificationService interface {
	NotifyBudgetReady(ctx context.Context, budget *entity.Budget) error
	NotifyPaymentRequested(ctx context.Context, budget *entity.Budget, payment *entity.Payment) error
}
