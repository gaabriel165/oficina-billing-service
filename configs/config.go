package configs

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort           string
	Environment          string
	ServiceName          string
	DatabaseURL          string
	JWTSecret            string
	AWSRegion            string
	SQSEndpoint          string
	BillingCommandsQueue string
	SagaRepliesQueue     string
	ResendAPIKey         string
	EmailFrom            string
	OtelExporterEndpoint string
	OtelExporterHeaders  string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	return &Config{
		ServerPort:           getEnv("SERVER_PORT", "8080"),
		Environment:          getEnv("APP_ENV", "local"),
		ServiceName:          getEnv("OTEL_SERVICE_NAME", "oficina-billing-service"),
		DatabaseURL:          getEnv("DATABASE_URL", ""),
		JWTSecret:            getEnv("JWT_SECRET", ""),
		AWSRegion:            getEnv("AWS_REGION", "us-east-1"),
		SQSEndpoint:          getEnv("SQS_ENDPOINT", ""),
		BillingCommandsQueue: getEnv("BILLING_COMMANDS_QUEUE", "oficina-billing-commands"),
		SagaRepliesQueue:     getEnv("SAGA_REPLIES_QUEUE", "oficina-saga-replies"),
		ResendAPIKey:         getEnv("RESEND_API_KEY", ""),
		EmailFrom:            getEnv("EMAIL_FROM", "onboarding@resend.dev"),
		OtelExporterEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		OtelExporterHeaders:  getEnv("OTEL_EXPORTER_OTLP_HEADERS", ""),
	}, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
