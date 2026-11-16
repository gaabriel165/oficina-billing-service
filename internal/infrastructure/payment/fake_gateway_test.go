package payment_test

import (
	"context"
	"testing"

	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
	"github.com/gabrielcamargo/oficina-billing-service/internal/infrastructure/payment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakeGateway_ShouldPointCheckoutToTheSimulationRoute(t *testing.T) {
	gateway := payment.NewFakeGateway("http://localhost:8081/")
	pending := entity.RestorePayment(entity.PaymentSnapshot{ID: "payment-1"})

	checkout, err := gateway.CreateCheckout(context.Background(), pending, nil)

	require.NoError(t, err)
	assert.Equal(t, "fake-payment-1", checkout.Reference)
	assert.Equal(t, "http://localhost:8081/api/v1/payments/payment-1/simulate", checkout.URL)
	assert.Equal(t, payment.ProviderFake, gateway.Provider())
}

func TestFakeGateway_ShouldAcceptRefunds(t *testing.T) {
	gateway := payment.NewFakeGateway("")

	assert.NoError(t, gateway.Refund(context.Background(), entity.RestorePayment(entity.PaymentSnapshot{ID: "payment-1"})))
}

func TestFakeGateway_ShouldNotResolveProviderPayments(t *testing.T) {
	gateway := payment.NewFakeGateway("")

	_, err := gateway.FindPayment(context.Background(), "provider-1")

	assert.ErrorIs(t, err, repository.ErrPaymentGatewayUnavailable)
}
