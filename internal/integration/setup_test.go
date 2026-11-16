package integration_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gabrielcamargo/oficina-billing-service/internal/infrastructure/database"
	"github.com/gabrielcamargo/oficina-billing-service/internal/infrastructure/messaging"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm"
)

const (
	elasticMQImage = "softwaremill/elasticmq-native:1.7.1"
	elasticMQPort  = "9324/tcp"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	dsn := fmt.Sprintf("postgres://test:test@%s:%s/testdb?sslmode=disable", host, port.Port())

	_, currentFile, _, _ := runtime.Caller(0)
	migrationsPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations")
	require.NoError(t, database.RunMigrations(dsn, migrationsPath))

	db, err := database.NewConnection(dsn)
	require.NoError(t, err)
	return db
}

func newTestQueueClient(t *testing.T, queueNames ...string) *sqs.Client {
	t.Helper()
	ctx := context.Background()
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        elasticMQImage,
			ExposedPorts: []string{elasticMQPort},
			WaitingFor:   wait.ForListeningPort(elasticMQPort),
		},
		Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, elasticMQPort)
	require.NoError(t, err)

	client, err := messaging.NewSQSClient(ctx, "us-east-1", fmt.Sprintf("http://%s:%s", host, port.Port()))
	require.NoError(t, err)

	for _, queueName := range queueNames {
		_, err := client.CreateQueue(ctx, &sqs.CreateQueueInput{QueueName: aws.String(queueName)})
		require.NoError(t, err)
	}
	return client
}
