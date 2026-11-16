package payment

import (
	"context"
	"errors"

	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
)

type RefundPaymentInput struct {
	OrderID string
	SagaID  string
}

type RefundPaymentUseCase struct {
	paymentRepo repository.PaymentRepository
	gateway     repository.PaymentGatewayService
	outbox      messaging.OutboxRepository
}

func NewRefundPaymentUseCase(
	paymentRepo repository.PaymentRepository,
	gateway repository.PaymentGatewayService,
	outbox messaging.OutboxRepository,
) *RefundPaymentUseCase {
	return &RefundPaymentUseCase{paymentRepo: paymentRepo, gateway: gateway, outbox: outbox}
}

func (u *RefundPaymentUseCase) Execute(ctx context.Context, input RefundPaymentInput) error {
	payment, err := u.paymentRepo.FindByOrderID(ctx, input.OrderID)
	if errors.Is(err, repository.ErrPaymentNotFound) {
		return u.acknowledge(ctx, input, "")
	}
	if err != nil {
		return err
	}

	if !payment.IsConfirmed() {
		return u.acknowledge(ctx, input, payment.ID())
	}

	if err := u.gateway.Refund(ctx, payment); err != nil {
		return err
	}

	if err := payment.Refund(); err != nil {
		return err
	}

	refunded, err := refundedReply(input, payment.ID())
	if err != nil {
		return err
	}
	return u.paymentRepo.Update(ctx, payment, refunded)
}

func (u *RefundPaymentUseCase) acknowledge(ctx context.Context, input RefundPaymentInput, paymentID string) error {
	refunded, err := refundedReply(input, paymentID)
	if err != nil {
		return err
	}
	return u.outbox.Enqueue(ctx, refunded)
}

func refundedReply(input RefundPaymentInput, paymentID string) (messaging.OutgoingMessage, error) {
	return usecase.Reply(messaging.TypePaymentRefunded, input.SagaID, input.OrderID, messaging.PaymentOutcomePayload{PaymentID: paymentID})
}
