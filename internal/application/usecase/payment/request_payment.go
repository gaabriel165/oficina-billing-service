package payment

import (
	"context"
	"errors"
	"log/slog"

	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
)

type RequestPaymentUseCase struct {
	budgetRepo    repository.BudgetRepository
	paymentRepo   repository.PaymentRepository
	gateway       repository.PaymentGatewayService
	notifications repository.NotificationService
}

func NewRequestPaymentUseCase(
	budgetRepo repository.BudgetRepository,
	paymentRepo repository.PaymentRepository,
	gateway repository.PaymentGatewayService,
	notifications repository.NotificationService,
) *RequestPaymentUseCase {
	return &RequestPaymentUseCase{budgetRepo: budgetRepo, paymentRepo: paymentRepo, gateway: gateway, notifications: notifications}
}

func (u *RequestPaymentUseCase) Execute(ctx context.Context, orderID string) (*entity.Payment, error) {
	existing, err := u.paymentRepo.FindByOrderID(ctx, orderID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, repository.ErrPaymentNotFound) {
		return nil, err
	}

	budget, err := u.budgetRepo.FindByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	payment, err := entity.NewPayment(budget, u.gateway.Provider())
	if err != nil {
		return nil, err
	}

	checkout, err := u.gateway.CreateCheckout(ctx, payment, budget)
	if err != nil {
		return nil, err
	}

	if err := payment.AttachCheckout(checkout.Reference, checkout.URL); err != nil {
		return nil, err
	}

	requested, err := usecase.Reply(messaging.TypePaymentRequested, payment.SagaID(), payment.OrderID(), messaging.PaymentRequestedPayload{
		PaymentID:   payment.ID(),
		CheckoutURL: payment.CheckoutURL(),
	})
	if err != nil {
		return nil, err
	}

	if err := u.paymentRepo.Create(ctx, payment, requested); err != nil {
		return nil, err
	}

	u.notify(ctx, budget, payment)
	return payment, nil
}

func (u *RequestPaymentUseCase) notify(ctx context.Context, budget *entity.Budget, payment *entity.Payment) {
	if err := u.notifications.NotifyPaymentRequested(ctx, budget, payment); err != nil {
		slog.ErrorContext(ctx, "integration.error",
			slog.String("event", "integration.error"),
			slog.String("integration", "resend"),
			slog.String("operation", "notify_payment_requested"),
			slog.String("order_id", payment.OrderID()),
			slog.String("error", err.Error()),
		)
	}
}
