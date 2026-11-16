package configs_test

import (
	"testing"

	"github.com/gabrielcamargo/oficina-billing-service/configs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_ShouldUseDefaultsWhenEnvironmentIsEmpty(t *testing.T) {
	t.Setenv("SERVER_PORT", "")
	t.Setenv("BILLING_COMMANDS_QUEUE", "")
	t.Setenv("SAGA_REPLIES_QUEUE", "")
	t.Setenv("OTEL_SERVICE_NAME", "")

	cfg, err := configs.Load()

	require.NoError(t, err)
	assert.Equal(t, "8080", cfg.ServerPort)
	assert.Equal(t, "oficina-billing-commands", cfg.BillingCommandsQueue)
	assert.Equal(t, "oficina-saga-replies", cfg.SagaRepliesQueue)
	assert.Equal(t, "oficina-billing-service", cfg.ServiceName)
}

func TestLoad_ShouldReadValuesFromEnvironment(t *testing.T) {
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("DATABASE_URL", "postgres://billing")
	t.Setenv("SQS_ENDPOINT", "http://localhost:9324")

	cfg, err := configs.Load()

	require.NoError(t, err)
	assert.Equal(t, "9090", cfg.ServerPort)
	assert.Equal(t, "postgres://billing", cfg.DatabaseURL)
	assert.Equal(t, "http://localhost:9324", cfg.SQSEndpoint)
}

func TestLoad_ShouldSelectMercadoPagoGateway(t *testing.T) {
	t.Setenv("PAYMENT_GATEWAY", "mercadopago")
	t.Setenv("MERCADOPAGO_ACCESS_TOKEN", "TEST-token")
	t.Setenv("MERCADOPAGO_WEBHOOK_SECRET", "webhook-secret")
	t.Setenv("MERCADOPAGO_BASE_URL", "")

	cfg, err := configs.Load()

	require.NoError(t, err)
	assert.True(t, cfg.MercadoPagoEnabled())
	assert.False(t, cfg.PaymentSimulationEnabled())
	assert.Equal(t, "TEST-token", cfg.MercadoPagoToken)
	assert.Equal(t, "webhook-secret", cfg.MercadoPagoSecret)
	assert.Equal(t, "https://api.mercadopago.com", cfg.MercadoPagoBaseURL)
}
