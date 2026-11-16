package repository

import (
	"context"

	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
)

type PaymentRepository interface {
	Create(ctx context.Context, payment *entity.Payment, outgoing ...messaging.OutgoingMessage) error
	Update(ctx context.Context, payment *entity.Payment, outgoing ...messaging.OutgoingMessage) error
	FindByID(ctx context.Context, id string) (*entity.Payment, error)
	FindByOrderID(ctx context.Context, orderID string) (*entity.Payment, error)
}
