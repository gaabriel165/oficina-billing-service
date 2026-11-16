package payment

import (
	"context"

	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
)

type GetPaymentUseCase struct {
	paymentRepo repository.PaymentRepository
}

func NewGetPaymentUseCase(paymentRepo repository.PaymentRepository) *GetPaymentUseCase {
	return &GetPaymentUseCase{paymentRepo: paymentRepo}
}

func (u *GetPaymentUseCase) Execute(ctx context.Context, paymentID string) (*entity.Payment, error) {
	return u.paymentRepo.FindByID(ctx, paymentID)
}

func (u *GetPaymentUseCase) ByOrder(ctx context.Context, orderID string) (*entity.Payment, error) {
	return u.paymentRepo.FindByOrderID(ctx, orderID)
}
