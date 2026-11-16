package consumer

import (
	"context"

	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase/budget"
	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase/payment"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
)

type Registrar interface {
	Register(messageType string, handler messaging.Handler)
}

type BillingConsumer struct {
	generateBudgetUC *budget.GenerateBudgetUseCase
	cancelBudgetUC   *budget.CancelBudgetUseCase
	requestPaymentUC *payment.RequestPaymentUseCase
	refundPaymentUC  *payment.RefundPaymentUseCase
}

func NewBillingConsumer(
	generateBudgetUC *budget.GenerateBudgetUseCase,
	cancelBudgetUC *budget.CancelBudgetUseCase,
	requestPaymentUC *payment.RequestPaymentUseCase,
	refundPaymentUC *payment.RefundPaymentUseCase,
) *BillingConsumer {
	return &BillingConsumer{
		generateBudgetUC: generateBudgetUC,
		cancelBudgetUC:   cancelBudgetUC,
		requestPaymentUC: requestPaymentUC,
		refundPaymentUC:  refundPaymentUC,
	}
}

func (c *BillingConsumer) RegisterHandlers(registrar Registrar) {
	registrar.Register(messaging.TypeGenerateBudget, c.GenerateBudget)
	registrar.Register(messaging.TypeCancelBudget, c.CancelBudget)
	registrar.Register(messaging.TypeRequestPayment, c.RequestPayment)
	registrar.Register(messaging.TypeRefundPayment, c.RefundPayment)
}

func (c *BillingConsumer) GenerateBudget(ctx context.Context, message messaging.Message) error {
	var payload messaging.GenerateBudgetPayload
	if err := message.DecodePayload(&payload); err != nil {
		return err
	}

	_, err := c.generateBudgetUC.Execute(ctx, budget.GenerateBudgetInput{
		OrderID:  message.OrderID,
		SagaID:   message.SagaID,
		Customer: entity.BudgetCustomer{ID: payload.CustomerID, Name: payload.CustomerName, Email: payload.CustomerEmail},
		Services: payload.Services,
		Parts:    payload.Parts,
	})
	return err
}

func (c *BillingConsumer) CancelBudget(ctx context.Context, message messaging.Message) error {
	return c.cancelBudgetUC.Execute(ctx, budget.CancelBudgetInput{OrderID: message.OrderID, SagaID: message.SagaID})
}

func (c *BillingConsumer) RequestPayment(ctx context.Context, message messaging.Message) error {
	_, err := c.requestPaymentUC.Execute(ctx, message.OrderID)
	return err
}

func (c *BillingConsumer) RefundPayment(ctx context.Context, message messaging.Message) error {
	return c.refundPaymentUC.Execute(ctx, payment.RefundPaymentInput{OrderID: message.OrderID, SagaID: message.SagaID})
}
