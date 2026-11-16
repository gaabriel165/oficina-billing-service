package integration_test

import (
	"context"
	"testing"

	"github.com/gabrielcamargo/oficina-billing-service/internal/application/outbox"
	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase"
	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase/budget"
	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase/payment"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/valueobject"
	infrapayment "github.com/gabrielcamargo/oficina-billing-service/internal/infrastructure/payment"
	infrarepo "github.com/gabrielcamargo/oficina-billing-service/internal/infrastructure/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type capturingPublisher struct {
	published []messaging.Message
}

func (p *capturingPublisher) Publish(_ context.Context, _ messaging.Destination, message messaging.Message) error {
	p.published = append(p.published, message)
	return nil
}

func (p *capturingPublisher) types() []string {
	types := make([]string, len(p.published))
	for index, message := range p.published {
		types[index] = message.Type
	}
	return types
}

type silentNotifier struct {
	budgetsNotified  int
	paymentsNotified int
}

func (n *silentNotifier) NotifyBudgetReady(context.Context, *entity.Budget) error {
	n.budgetsNotified++
	return nil
}

func (n *silentNotifier) NotifyPaymentRequested(context.Context, *entity.Budget, *entity.Payment) error {
	n.paymentsNotified++
	return nil
}

type billingFixture struct {
	ctx        context.Context
	budgets    *infrarepo.GormBudgetRepository
	payments   *infrarepo.GormPaymentRepository
	outboxRepo *infrarepo.GormOutboxRepository
	relay      *outbox.Relay
	publisher  *capturingPublisher
	notifier   *silentNotifier
	gateway    repository.PaymentGatewayService
	orderID    string
	sagaID     string
	customerID string
	operator   budget.Requester
}

func newBillingFixture(t *testing.T) *billingFixture {
	t.Helper()
	db := newTestDB(t)
	fixture := &billingFixture{
		ctx:        context.Background(),
		budgets:    infrarepo.NewGormBudgetRepository(db),
		payments:   infrarepo.NewGormPaymentRepository(db),
		outboxRepo: infrarepo.NewGormOutboxRepository(db),
		publisher:  &capturingPublisher{},
		notifier:   &silentNotifier{},
		gateway:    infrapayment.NewFakeGateway("http://localhost:8081"),
		orderID:    uuid.NewString(),
		sagaID:     uuid.NewString(),
		customerID: uuid.NewString(),
		operator:   budget.Requester{ID: uuid.NewString()},
	}
	fixture.relay = outbox.NewRelay(fixture.outboxRepo, fixture.publisher)
	return fixture
}

func (f *billingFixture) generate(t *testing.T) *entity.Budget {
	t.Helper()
	generated, err := budget.NewGenerateBudgetUseCase(f.budgets, f.notifier).Execute(f.ctx, budget.GenerateBudgetInput{
		OrderID:  f.orderID,
		SagaID:   f.sagaID,
		Customer: entity.BudgetCustomer{ID: f.customerID, Name: "Maria Silva", Email: "maria@example.com"},
		Services: []messaging.ServiceLine{{ServiceID: uuid.NewString(), Name: "Oil change", LaborPrice: 120}},
		Parts:    []messaging.PartLine{{PartID: uuid.NewString(), Name: "Oil filter", Quantity: 2, UnitPrice: 29.90}},
	})
	require.NoError(t, err)
	return generated
}

func (f *billingFixture) approveAndRequestPayment(t *testing.T) *entity.Payment {
	t.Helper()
	generated := f.generate(t)
	_, err := budget.NewApproveBudgetUseCase(f.budgets).Execute(f.ctx, generated.ID(), f.operator)
	require.NoError(t, err)

	requested, err := payment.NewRequestPaymentUseCase(f.budgets, f.payments, f.gateway, f.notifier).Execute(f.ctx, f.orderID)
	require.NoError(t, err)
	return requested
}

func (f *billingFixture) relayAll(t *testing.T) {
	t.Helper()
	_, err := f.relay.RelayOnce(f.ctx)
	require.NoError(t, err)
}

func TestBillingFlow_ShouldGenerateBudgetWithItemsAndTotal(t *testing.T) {
	fixture := newBillingFixture(t)

	generated := fixture.generate(t)
	stored, err := fixture.budgets.FindByOrderID(fixture.ctx, fixture.orderID)
	require.NoError(t, err)
	fixture.relayAll(t)

	assert.Equal(t, generated.ID(), stored.ID())
	assert.Len(t, stored.Items(), 2)
	assert.InDelta(t, 179.80, stored.TotalAmount(), 0.001)
	assert.Equal(t, valueobject.BudgetStatusPendingApproval, stored.Status())
	assert.Equal(t, []string{messaging.TypeBudgetGenerated}, fixture.publisher.types())
	assert.Equal(t, 1, fixture.notifier.budgetsNotified)
}

func TestBillingFlow_ShouldNotDuplicateBudgetWhenCommandIsRedelivered(t *testing.T) {
	fixture := newBillingFixture(t)

	first := fixture.generate(t)
	second := fixture.generate(t)
	fixture.relayAll(t)

	assert.Equal(t, first.ID(), second.ID())
	assert.Equal(t, []string{messaging.TypeBudgetGenerated}, fixture.publisher.types())
}

func TestBillingFlow_ShouldConfirmPaymentAfterApproval(t *testing.T) {
	fixture := newBillingFixture(t)
	requested := fixture.approveAndRequestPayment(t)

	settled, err := payment.NewSettlePaymentUseCase(fixture.payments).Execute(fixture.ctx, payment.SettlePaymentInput{
		PaymentID: requested.ID(), ProviderPaymentID: "provider-1", Approved: true,
	})
	require.NoError(t, err)
	fixture.relayAll(t)

	assert.Equal(t, valueobject.PaymentStatusConfirmed, settled.Status())
	assert.InDelta(t, 179.80, settled.Amount(), 0.001)
	assert.Contains(t, requested.CheckoutURL(), requested.ID())
	assert.Equal(t, []string{
		messaging.TypeBudgetGenerated,
		messaging.TypeBudgetApproved,
		messaging.TypePaymentRequested,
		messaging.TypePaymentConfirmed,
	}, fixture.publisher.types())
}

func TestBillingFlow_ShouldReportFailedPayment(t *testing.T) {
	fixture := newBillingFixture(t)
	requested := fixture.approveAndRequestPayment(t)

	settled, err := payment.NewSettlePaymentUseCase(fixture.payments).Execute(fixture.ctx, payment.SettlePaymentInput{
		PaymentID: requested.ID(), Approved: false, Reason: "card declined",
	})
	require.NoError(t, err)
	fixture.relayAll(t)

	assert.Equal(t, valueobject.PaymentStatusFailed, settled.Status())
	assert.Equal(t, "card declined", settled.FailureReason())
	assert.Contains(t, fixture.publisher.types(), messaging.TypePaymentFailed)
}

func TestBillingFlow_ShouldIgnoreSecondSettlement(t *testing.T) {
	fixture := newBillingFixture(t)
	requested := fixture.approveAndRequestPayment(t)
	settle := payment.NewSettlePaymentUseCase(fixture.payments)

	_, err := settle.Execute(fixture.ctx, payment.SettlePaymentInput{PaymentID: requested.ID(), Approved: true})
	require.NoError(t, err)
	again, err := settle.Execute(fixture.ctx, payment.SettlePaymentInput{PaymentID: requested.ID(), Approved: false})
	require.NoError(t, err)

	assert.Equal(t, valueobject.PaymentStatusConfirmed, again.Status())
}

func TestBillingFlow_ShouldRefundConfirmedPaymentAndCancelBudget(t *testing.T) {
	fixture := newBillingFixture(t)
	requested := fixture.approveAndRequestPayment(t)
	_, err := payment.NewSettlePaymentUseCase(fixture.payments).Execute(fixture.ctx, payment.SettlePaymentInput{
		PaymentID: requested.ID(), Approved: true,
	})
	require.NoError(t, err)

	require.NoError(t, payment.NewRefundPaymentUseCase(fixture.payments, fixture.gateway, fixture.outboxRepo).Execute(fixture.ctx, payment.RefundPaymentInput{
		OrderID: fixture.orderID, SagaID: fixture.sagaID,
	}))
	require.NoError(t, budget.NewCancelBudgetUseCase(fixture.budgets, fixture.outboxRepo).Execute(fixture.ctx, budget.CancelBudgetInput{
		OrderID: fixture.orderID, SagaID: fixture.sagaID,
	}))
	fixture.relayAll(t)

	refunded, err := fixture.payments.FindByOrderID(fixture.ctx, fixture.orderID)
	require.NoError(t, err)
	cancelled, err := fixture.budgets.FindByOrderID(fixture.ctx, fixture.orderID)
	require.NoError(t, err)
	assert.Equal(t, valueobject.PaymentStatusRefunded, refunded.Status())
	assert.Equal(t, valueobject.BudgetStatusCancelled, cancelled.Status())
	assert.Contains(t, fixture.publisher.types(), messaging.TypePaymentRefunded)
	assert.Contains(t, fixture.publisher.types(), messaging.TypeBudgetCancelled)
}

func TestBillingFlow_ShouldAnswerCompensationsWhenNothingExists(t *testing.T) {
	fixture := newBillingFixture(t)

	require.NoError(t, payment.NewRefundPaymentUseCase(fixture.payments, fixture.gateway, fixture.outboxRepo).Execute(fixture.ctx, payment.RefundPaymentInput{
		OrderID: fixture.orderID, SagaID: fixture.sagaID,
	}))
	require.NoError(t, budget.NewCancelBudgetUseCase(fixture.budgets, fixture.outboxRepo).Execute(fixture.ctx, budget.CancelBudgetInput{
		OrderID: fixture.orderID, SagaID: fixture.sagaID,
	}))
	fixture.relayAll(t)

	assert.ElementsMatch(t, []string{messaging.TypePaymentRefunded, messaging.TypeBudgetCancelled}, fixture.publisher.types())
}

func TestBillingFlow_ShouldRejectBudgetWithReason(t *testing.T) {
	fixture := newBillingFixture(t)
	generated := fixture.generate(t)

	rejected, err := budget.NewRejectBudgetUseCase(fixture.budgets).Execute(fixture.ctx, budget.RejectBudgetInput{
		BudgetID: generated.ID(), Reason: "too expensive", Requester: budget.Requester{ID: fixture.customerID, IsCustomer: true},
	})
	require.NoError(t, err)
	fixture.relayAll(t)

	assert.Equal(t, valueobject.BudgetStatusRejected, rejected.Status())
	assert.Contains(t, fixture.publisher.types(), messaging.TypeBudgetRejected)
}

func TestBillingFlow_ShouldForbidCustomerFromDecidingAnotherCustomersBudget(t *testing.T) {
	fixture := newBillingFixture(t)
	generated := fixture.generate(t)
	stranger := budget.Requester{ID: uuid.NewString(), IsCustomer: true}

	_, err := budget.NewApproveBudgetUseCase(fixture.budgets).Execute(fixture.ctx, generated.ID(), stranger)

	assert.ErrorIs(t, err, usecase.ErrForbidden)
}

func TestBillingFlow_ShouldNotRequestPaymentBeforeApproval(t *testing.T) {
	fixture := newBillingFixture(t)
	fixture.generate(t)

	_, err := payment.NewRequestPaymentUseCase(fixture.budgets, fixture.payments, fixture.gateway, fixture.notifier).Execute(fixture.ctx, fixture.orderID)

	assert.ErrorIs(t, err, entity.ErrBudgetNotApproved)
}

func TestBudgetRepository_ShouldListByCustomerAndReportMissing(t *testing.T) {
	fixture := newBillingFixture(t)
	fixture.generate(t)

	mine, err := fixture.budgets.FindByCustomerID(fixture.ctx, fixture.customerID)
	require.NoError(t, err)
	others, err := fixture.budgets.FindByCustomerID(fixture.ctx, uuid.NewString())
	require.NoError(t, err)
	all, err := fixture.budgets.FindAll(fixture.ctx)
	require.NoError(t, err)
	_, missingBudget := fixture.budgets.FindByID(fixture.ctx, uuid.NewString())
	_, missingPayment := fixture.payments.FindByID(fixture.ctx, uuid.NewString())

	assert.Len(t, mine, 1)
	assert.Empty(t, others)
	assert.Len(t, all, 1)
	assert.ErrorIs(t, missingBudget, repository.ErrBudgetNotFound)
	assert.ErrorIs(t, missingPayment, repository.ErrPaymentNotFound)
}
