package api_test

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gabrielcamargo/oficina-billing-service/configs"
	"github.com/gabrielcamargo/oficina-billing-service/internal/api"
	"github.com/gabrielcamargo/oficina-billing-service/internal/application/mocks"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/valueobject"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testJWTSecret     = "test-secret"
	testWebhookSecret = "webhook-secret"
	roleOperator      = "operator"
	roleCustomer      = "customer"
)

type testEnvironment struct {
	server        *api.Server
	budgetRepo    *mocks.MockBudgetRepository
	paymentRepo   *mocks.MockPaymentRepository
	outbox        *mocks.MockOutboxRepository
	gateway       *mocks.MockPaymentGatewayService
	notifications *mocks.MockNotificationService
}

func newTestEnvironment(t *testing.T, paymentGateway string) *testEnvironment {
	t.Helper()
	gin.SetMode(gin.TestMode)
	environment := &testEnvironment{
		budgetRepo:    mocks.NewMockBudgetRepository(t),
		paymentRepo:   mocks.NewMockPaymentRepository(t),
		outbox:        mocks.NewMockOutboxRepository(t),
		gateway:       mocks.NewMockPaymentGatewayService(t),
		notifications: mocks.NewMockNotificationService(t),
	}
	environment.server = api.NewServer(
		&configs.Config{ServerPort: "0", ServiceName: "oficina-billing-service", JWTSecret: testJWTSecret, PaymentGateway: paymentGateway, MercadoPagoSecret: testWebhookSecret},
		api.Dependencies{
			BudgetRepo:    environment.budgetRepo,
			PaymentRepo:   environment.paymentRepo,
			Outbox:        environment.outbox,
			Gateway:       environment.gateway,
			Notifications: environment.notifications,
		},
	)
	return environment
}

func (e *testEnvironment) request(t *testing.T, method, path, role, subject, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if role != "" {
		request.Header.Set("Authorization", "Bearer "+tokenFor(t, subject, role))
	}
	recorder := httptest.NewRecorder()
	e.server.Handler().ServeHTTP(recorder, request)
	return recorder
}

func tokenFor(t *testing.T, subject, role string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  subject,
		"role": role,
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(testJWTSecret))
	require.NoError(t, err)
	return signed
}

func pendingBudget(t *testing.T) *entity.Budget {
	t.Helper()
	item, err := entity.NewBudgetItem(valueobject.BudgetItemKindService, "service-1", "Troca de óleo", 1, 120)
	require.NoError(t, err)
	created, err := entity.NewBudget("order-1", "saga-1", entity.BudgetCustomer{ID: "customer-1", Name: "Maria Silva", Email: "maria@example.com"}, []*entity.BudgetItem{item})
	require.NoError(t, err)
	return created
}

func pendingPayment(t *testing.T) *entity.Payment {
	t.Helper()
	budget := pendingBudget(t)
	require.NoError(t, budget.Approve())
	created, err := entity.NewPayment(budget, "fake")
	require.NoError(t, err)
	require.NoError(t, created.AttachCheckout("pref-1", "https://checkout.example.com/pref-1"))
	return created
}

func replyOfType(messageType string) any {
	return mock.MatchedBy(func(outgoing []messaging.OutgoingMessage) bool {
		return len(outgoing) == 1 && outgoing[0].Message.Type == messageType
	})
}

func assertStatus(t *testing.T, expected int, recorder *httptest.ResponseRecorder) {
	t.Helper()
	require.Equal(t, expected, recorder.Code, recorder.Body.String())
}
