package payment

import (
	"context"

	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
)

const defaultFailureReason = "payment was not approved"

type SettlePaymentInput struct {
	PaymentID         string
	ProviderPaymentID string
	Approved          bool
	Reason            string
}

type SettlePaymentUseCase struct {
	paymentRepo repository.PaymentRepository
}

func NewSettlePaymentUseCase(paymentRepo repository.PaymentRepository) *SettlePaymentUseCase {
	return &SettlePaymentUseCase{paymentRepo: paymentRepo}
}

func (u *SettlePaymentUseCase) Execute(ctx context.Context, input SettlePaymentInput) (*entity.Payment, error) {
	payment, err := u.paymentRepo.FindByID(ctx, input.PaymentID)
	if err != nil {
		return nil, err
	}

	if !payment.IsPending() {
		return payment, nil
	}

	outcome, err := settle(payment, input)
	if err != nil {
		return nil, err
	}

	if err := u.paymentRepo.Update(ctx, payment, outcome); err != nil {
		return nil, err
	}
	return payment, nil
}

func settle(payment *entity.Payment, input SettlePaymentInput) (messaging.OutgoingMessage, error) {
	if input.Approved {
		if err := payment.Confirm(input.ProviderPaymentID); err != nil {
			return messaging.OutgoingMessage{}, err
		}
		return usecase.Reply(messaging.TypePaymentConfirmed, payment.SagaID(), payment.OrderID(), messaging.PaymentOutcomePayload{
			PaymentID: payment.ID(),
		})
	}

	reason := input.Reason
	if reason == "" {
		reason = defaultFailureReason
	}
	if err := payment.Fail(input.ProviderPaymentID, reason); err != nil {
		return messaging.OutgoingMessage{}, err
	}
	return usecase.Reply(messaging.TypePaymentFailed, payment.SagaID(), payment.OrderID(), messaging.PaymentOutcomePayload{
		PaymentID: payment.ID(),
		Reason:    reason,
	})
}
