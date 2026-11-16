package payment_test

import (
	"testing"

	"github.com/gabrielcamargo/oficina-billing-service/internal/application/mocks"
	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase/payment"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/valueobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var refundInput = payment.RefundPaymentInput{OrderID: "order-1", SagaID: "saga-1"}

type refundPaymentMocks struct {
	paymentRepo *mocks.MockPaymentRepository
	gateway     *mocks.MockPaymentGatewayService
	outbox      *mocks.MockOutboxRepository
}

func newRefundPaymentMocks(t *testing.T) refundPaymentMocks {
	return refundPaymentMocks{
		paymentRepo: mocks.NewMockPaymentRepository(t),
		gateway:     mocks.NewMockPaymentGatewayService(t),
		outbox:      mocks.NewMockOutboxRepository(t),
	}
}

func (m refundPaymentMocks) useCase() *payment.RefundPaymentUseCase {
	return payment.NewRefundPaymentUseCase(m.paymentRepo, m.gateway, m.outbox)
}

func TestRefundPayment_ShouldRefundConfirmedPaymentAndReplyToSaga(t *testing.T) {
	m := newRefundPaymentMocks(t)
	existing := confirmedPayment(t)
	m.paymentRepo.On("FindByOrderID", ctx, "order-1").Return(existing, nil)
	m.gateway.On("Refund", ctx, existing).Return(nil)
	m.paymentRepo.On("Update", ctx, existing, replyOfType(messaging.TypePaymentRefunded)).Return(nil)

	err := m.useCase().Execute(ctx, refundInput)

	require.NoError(t, err)
	assert.Equal(t, valueobject.PaymentStatusRefunded, existing.Status())
}

func TestRefundPayment_ShouldOnlyReplyWhenPaymentDoesNotExist(t *testing.T) {
	m := newRefundPaymentMocks(t)
	m.paymentRepo.On("FindByOrderID", ctx, "order-1").Return(nil, repository.ErrPaymentNotFound)
	m.outbox.On("Enqueue", ctx, replyOfType(messaging.TypePaymentRefunded)).Return(nil)

	err := m.useCase().Execute(ctx, refundInput)

	assert.NoError(t, err)
}

func TestRefundPayment_ShouldOnlyReplyWhenPaymentWasNeverConfirmed(t *testing.T) {
	m := newRefundPaymentMocks(t)
	existing := pendingPayment(t)
	m.paymentRepo.On("FindByOrderID", ctx, "order-1").Return(existing, nil)
	m.outbox.On("Enqueue", ctx, replyOfType(messaging.TypePaymentRefunded)).Return(nil)

	err := m.useCase().Execute(ctx, refundInput)

	require.NoError(t, err)
	assert.True(t, existing.IsPending())
}

func TestRefundPayment_ShouldKeepPaymentConfirmedWhenGatewayFails(t *testing.T) {
	m := newRefundPaymentMocks(t)
	existing := confirmedPayment(t)
	m.paymentRepo.On("FindByOrderID", ctx, "order-1").Return(existing, nil)
	m.gateway.On("Refund", ctx, existing).Return(repository.ErrPaymentGatewayUnavailable)

	err := m.useCase().Execute(ctx, refundInput)

	assert.ErrorIs(t, err, repository.ErrPaymentGatewayUnavailable)
	assert.True(t, existing.IsConfirmed())
}

func TestRefundPayment_ShouldReturnErrorWhenLookupFails(t *testing.T) {
	m := newRefundPaymentMocks(t)
	m.paymentRepo.On("FindByOrderID", ctx, "order-1").Return(nil, errDatabase)

	err := m.useCase().Execute(ctx, refundInput)

	assert.ErrorIs(t, err, errDatabase)
}

func TestRefundPayment_ShouldReturnErrorWhenSagaIsMissing(t *testing.T) {
	m := newRefundPaymentMocks(t)
	m.paymentRepo.On("FindByOrderID", ctx, "order-1").Return(nil, repository.ErrPaymentNotFound)

	err := m.useCase().Execute(ctx, payment.RefundPaymentInput{OrderID: "order-1"})

	assert.ErrorIs(t, err, messaging.ErrMessageSagaRequired)
}
