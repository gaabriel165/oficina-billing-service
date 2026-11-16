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

func TestSettlePayment_ShouldConfirmApprovedPayment(t *testing.T) {
	paymentRepo := mocks.NewMockPaymentRepository(t)
	existing := pendingPayment(t)
	paymentRepo.On("FindByID", ctx, existing.ID()).Return(existing, nil)
	paymentRepo.On("Update", ctx, existing, replyOfType(messaging.TypePaymentConfirmed)).Return(nil)

	result, err := payment.NewSettlePaymentUseCase(paymentRepo).Execute(ctx, payment.SettlePaymentInput{
		PaymentID: existing.ID(), ProviderPaymentID: "mp-123", Approved: true,
	})

	require.NoError(t, err)
	assert.True(t, result.IsConfirmed())
	assert.Equal(t, "mp-123", result.ProviderPaymentID())
}

func TestSettlePayment_ShouldFailRejectedPaymentWithGatewayReason(t *testing.T) {
	paymentRepo := mocks.NewMockPaymentRepository(t)
	existing := pendingPayment(t)
	paymentRepo.On("FindByID", ctx, existing.ID()).Return(existing, nil)
	paymentRepo.On("Update", ctx, existing, replyOfType(messaging.TypePaymentFailed)).Return(nil)

	result, err := payment.NewSettlePaymentUseCase(paymentRepo).Execute(ctx, payment.SettlePaymentInput{
		PaymentID: existing.ID(), ProviderPaymentID: "mp-123", Reason: "cc_rejected_other_reason",
	})

	require.NoError(t, err)
	assert.Equal(t, valueobject.PaymentStatusFailed, result.Status())
	assert.Equal(t, "cc_rejected_other_reason", result.FailureReason())
}

func TestSettlePayment_ShouldUseDefaultReasonWhenGatewayGivesNone(t *testing.T) {
	paymentRepo := mocks.NewMockPaymentRepository(t)
	existing := pendingPayment(t)
	paymentRepo.On("FindByID", ctx, existing.ID()).Return(existing, nil)
	paymentRepo.On("Update", ctx, existing, replyOfType(messaging.TypePaymentFailed)).Return(nil)

	result, err := payment.NewSettlePaymentUseCase(paymentRepo).Execute(ctx, payment.SettlePaymentInput{PaymentID: existing.ID()})

	require.NoError(t, err)
	assert.Equal(t, "payment was not approved", result.FailureReason())
}

func TestSettlePayment_ShouldIgnoreNotificationForSettledPayment(t *testing.T) {
	paymentRepo := mocks.NewMockPaymentRepository(t)
	existing := confirmedPayment(t)
	paymentRepo.On("FindByID", ctx, existing.ID()).Return(existing, nil)

	result, err := payment.NewSettlePaymentUseCase(paymentRepo).Execute(ctx, payment.SettlePaymentInput{PaymentID: existing.ID()})

	require.NoError(t, err)
	assert.True(t, result.IsConfirmed())
}

func TestSettlePayment_ShouldReturnErrorWhenNotFound(t *testing.T) {
	paymentRepo := mocks.NewMockPaymentRepository(t)
	paymentRepo.On("FindByID", ctx, "missing").Return(nil, repository.ErrPaymentNotFound)

	_, err := payment.NewSettlePaymentUseCase(paymentRepo).Execute(ctx, payment.SettlePaymentInput{PaymentID: "missing"})

	assert.ErrorIs(t, err, repository.ErrPaymentNotFound)
}

func TestSettlePayment_ShouldReturnErrorWhenSaveFails(t *testing.T) {
	paymentRepo := mocks.NewMockPaymentRepository(t)
	existing := pendingPayment(t)
	paymentRepo.On("FindByID", ctx, existing.ID()).Return(existing, nil)
	paymentRepo.On("Update", ctx, existing, replyOfType(messaging.TypePaymentConfirmed)).Return(errDatabase)

	_, err := payment.NewSettlePaymentUseCase(paymentRepo).Execute(ctx, payment.SettlePaymentInput{PaymentID: existing.ID(), Approved: true})

	assert.ErrorIs(t, err, errDatabase)
}
