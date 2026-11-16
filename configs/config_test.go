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
