package repository

import (
	"context"

	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
)

type BudgetRepository interface {
	Create(ctx context.Context, budget *entity.Budget, outgoing ...messaging.OutgoingMessage) error
	Update(ctx context.Context, budget *entity.Budget, outgoing ...messaging.OutgoingMessage) error
	FindByID(ctx context.Context, id string) (*entity.Budget, error)
	FindByOrderID(ctx context.Context, orderID string) (*entity.Budget, error)
	FindAll(ctx context.Context) ([]*entity.Budget, error)
	FindByCustomerID(ctx context.Context, customerID string) ([]*entity.Budget, error)
}
