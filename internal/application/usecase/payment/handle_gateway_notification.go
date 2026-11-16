package payment

import (
	"context"

	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
)

type HandleGatewayNotificationUseCase struct {
	gateway  repository.PaymentGatewayService
	settleUC *SettlePaymentUseCase
}

func NewHandleGatewayNotificationUseCase(
	gateway repository.PaymentGatewayService,
	settleUC *SettlePaymentUseCase,
) *HandleGatewayNotificationUseCase {
	return &HandleGatewayNotificationUseCase{gateway: gateway, settleUC: settleUC}
}

func (u *HandleGatewayNotificationUseCase) Execute(ctx context.Context, providerPaymentID string) (*entity.Payment, error) {
	gatewayPayment, err := u.gateway.FindPayment(ctx, providerPaymentID)
	if err != nil {
		return nil, err
	}

	if gatewayPayment.Status == repository.GatewayPaymentPending {
		return nil, nil
	}

	return u.settleUC.Execute(ctx, SettlePaymentInput{
		PaymentID:         gatewayPayment.PaymentID,
		ProviderPaymentID: gatewayPayment.ProviderPaymentID,
		Approved:          gatewayPayment.Status == repository.GatewayPaymentApproved,
		Reason:            gatewayPayment.StatusDetail,
	})
}
