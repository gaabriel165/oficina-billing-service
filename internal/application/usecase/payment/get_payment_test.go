package payment_test

import (
	"testing"

	"github.com/gabrielcamargo/oficina-billing-service/internal/application/mocks"
	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase/payment"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPayment_ShouldReturnPaymentByID(t *testing.T) {
	paymentRepo := mocks.NewMockPaymentRepository(t)
	existing := pendingPayment(t)
	paymentRepo.On("FindByID", ctx, existing.ID()).Return(existing, nil)

	result, err := payment.NewGetPaymentUseCase(paymentRepo).Execute(ctx, existing.ID())

	require.NoError(t, err)
	assert.Equal(t, existing.ID(), result.ID())
}

func TestGetPayment_ShouldReturnErrorWhenNotFound(t *testing.T) {
	paymentRepo := mocks.NewMockPaymentRepository(t)
	paymentRepo.On("FindByID", ctx, "missing").Return(nil, repository.ErrPaymentNotFound)

	_, err := payment.NewGetPaymentUseCase(paymentRepo).Execute(ctx, "missing")

	assert.ErrorIs(t, err, repository.ErrPaymentNotFound)
}

func TestGetPaymentByOrder_ShouldReturnPaymentOfOrder(t *testing.T) {
	paymentRepo := mocks.NewMockPaymentRepository(t)
	existing := pendingPayment(t)
	paymentRepo.On("FindByOrderID", ctx, "order-1").Return(existing, nil)

	result, err := payment.NewGetPaymentUseCase(paymentRepo).ByOrder(ctx, "order-1")

	require.NoError(t, err)
	assert.Equal(t, "order-1", result.OrderID())
}
