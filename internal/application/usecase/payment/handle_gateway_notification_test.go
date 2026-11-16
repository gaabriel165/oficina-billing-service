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

func TestHandleGatewayNotification_ShouldConfirmApprovedPayment(t *testing.T) {
	paymentRepo := mocks.NewMockPaymentRepository(t)
	gateway := mocks.NewMockPaymentGatewayService(t)
	existing := pendingPayment(t)
	gateway.On("FindPayment", ctx, "mp-123").Return(repository.GatewayPayment{
		ProviderPaymentID: "mp-123", PaymentID: existing.ID(), Status: repository.GatewayPaymentApproved,
	}, nil)
	paymentRepo.On("FindByID", ctx, existing.ID()).Return(existing, nil)
	paymentRepo.On("Update", ctx, existing, replyOfType(messaging.TypePaymentConfirmed)).Return(nil)

	result, err := payment.NewHandleGatewayNotificationUseCase(gateway, payment.NewSettlePaymentUseCase(paymentRepo)).Execute(ctx, "mp-123")

	require.NoError(t, err)
	assert.True(t, result.IsConfirmed())
}

func TestHandleGatewayNotification_ShouldFailRejectedPayment(t *testing.T) {
	paymentRepo := mocks.NewMockPaymentRepository(t)
	gateway := mocks.NewMockPaymentGatewayService(t)
	existing := pendingPayment(t)
	gateway.On("FindPayment", ctx, "mp-123").Return(repository.GatewayPayment{
		ProviderPaymentID: "mp-123", PaymentID: existing.ID(), Status: repository.GatewayPaymentRejected, StatusDetail: "cc_rejected_bad_filled_date",
	}, nil)
	paymentRepo.On("FindByID", ctx, existing.ID()).Return(existing, nil)
	paymentRepo.On("Update", ctx, existing, replyOfType(messaging.TypePaymentFailed)).Return(nil)

	result, err := payment.NewHandleGatewayNotificationUseCase(gateway, payment.NewSettlePaymentUseCase(paymentRepo)).Execute(ctx, "mp-123")

	require.NoError(t, err)
	assert.Equal(t, valueobject.PaymentStatusFailed, result.Status())
	assert.Equal(t, "cc_rejected_bad_filled_date", result.FailureReason())
}

func TestHandleGatewayNotification_ShouldIgnorePendingPayment(t *testing.T) {
	paymentRepo := mocks.NewMockPaymentRepository(t)
	gateway := mocks.NewMockPaymentGatewayService(t)
	gateway.On("FindPayment", ctx, "mp-123").Return(repository.GatewayPayment{Status: repository.GatewayPaymentPending}, nil)

	result, err := payment.NewHandleGatewayNotificationUseCase(gateway, payment.NewSettlePaymentUseCase(paymentRepo)).Execute(ctx, "mp-123")

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestHandleGatewayNotification_ShouldReturnErrorWhenGatewayIsUnavailable(t *testing.T) {
	paymentRepo := mocks.NewMockPaymentRepository(t)
	gateway := mocks.NewMockPaymentGatewayService(t)
	gateway.On("FindPayment", ctx, "mp-123").Return(repository.GatewayPayment{}, repository.ErrPaymentGatewayUnavailable)

	_, err := payment.NewHandleGatewayNotificationUseCase(gateway, payment.NewSettlePaymentUseCase(paymentRepo)).Execute(ctx, "mp-123")

	assert.ErrorIs(t, err, repository.ErrPaymentGatewayUnavailable)
}
