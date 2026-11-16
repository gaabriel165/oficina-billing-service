package api_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gabrielcamargo/oficina-billing-service/configs"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	webhookTimestamp    = "1704908010"
	webhookRequestID    = "req-1"
	providerPaymentID   = "999999999"
	paymentNotice       = `{"type":"payment","action":"payment.updated","data":{"id":"999999999"}}`
	merchantOrderNotice = `{"type":"topic_merchant_order_wh","action":"merchant_order.updated","data":{"id":"999999999"}}`
)

func signedWebhook(environment *testEnvironment, secret, body string) *httptest.ResponseRecorder {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("id:" + providerPaymentID + ";request-id:" + webhookRequestID + ";ts:" + webhookTimestamp + ";"))

	request := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/mercadopago?data.id="+providerPaymentID+"&type=payment", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("x-request-id", webhookRequestID)
	request.Header.Set("x-signature", "ts="+webhookTimestamp+",v1="+hex.EncodeToString(mac.Sum(nil)))
	recorder := httptest.NewRecorder()
	environment.server.Handler().ServeHTTP(recorder, request)
	return recorder
}

func TestMercadoPagoWebhook_ShouldConfirmApprovedPayment(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayMercadoPago)
	payment := pendingPayment(t)
	environment.gateway.On("FindPayment", mock.Anything, providerPaymentID).Return(repository.GatewayPayment{
		ProviderPaymentID: providerPaymentID, PaymentID: payment.ID(), Status: repository.GatewayPaymentApproved,
	}, nil)
	environment.paymentRepo.On("FindByID", mock.Anything, payment.ID()).Return(payment, nil)
	environment.paymentRepo.On("Update", mock.Anything, payment, replyOfType(messaging.TypePaymentConfirmed)).Return(nil)

	recorder := signedWebhook(environment, testWebhookSecret, paymentNotice)

	assertStatus(t, http.StatusOK, recorder)
	assert.JSONEq(t, `{"status":"processed"}`, recorder.Body.String())
	assert.True(t, payment.IsConfirmed())
	assert.Equal(t, providerPaymentID, payment.ProviderPaymentID())
}

func TestMercadoPagoWebhook_ShouldFailRejectedPayment(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayMercadoPago)
	payment := pendingPayment(t)
	environment.gateway.On("FindPayment", mock.Anything, providerPaymentID).Return(repository.GatewayPayment{
		ProviderPaymentID: providerPaymentID, PaymentID: payment.ID(), Status: repository.GatewayPaymentRejected, StatusDetail: "cc_rejected_other_reason",
	}, nil)
	environment.paymentRepo.On("FindByID", mock.Anything, payment.ID()).Return(payment, nil)
	environment.paymentRepo.On("Update", mock.Anything, payment, replyOfType(messaging.TypePaymentFailed)).Return(nil)

	recorder := signedWebhook(environment, testWebhookSecret, paymentNotice)

	assertStatus(t, http.StatusOK, recorder)
	assert.Equal(t, "cc_rejected_other_reason", payment.FailureReason())
}

func TestMercadoPagoWebhook_ShouldRejectInvalidSignature(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayMercadoPago)

	recorder := signedWebhook(environment, "another-secret", paymentNotice)

	assertStatus(t, http.StatusUnauthorized, recorder)
}

func TestMercadoPagoWebhook_ShouldIgnoreOtherNotificationTypes(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayMercadoPago)

	recorder := signedWebhook(environment, testWebhookSecret, merchantOrderNotice)

	assertStatus(t, http.StatusOK, recorder)
	assert.JSONEq(t, `{"status":"ignored"}`, recorder.Body.String())
}

func TestMercadoPagoWebhook_ShouldIgnorePendingPayment(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayMercadoPago)
	environment.gateway.On("FindPayment", mock.Anything, providerPaymentID).Return(repository.GatewayPayment{Status: repository.GatewayPaymentPending}, nil)

	recorder := signedWebhook(environment, testWebhookSecret, paymentNotice)

	assertStatus(t, http.StatusOK, recorder)
	assert.JSONEq(t, `{"status":"ignored"}`, recorder.Body.String())
}

func TestMercadoPagoWebhook_ShouldIgnorePaymentFromAnotherIntegration(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayMercadoPago)
	environment.gateway.On("FindPayment", mock.Anything, providerPaymentID).Return(repository.GatewayPayment{
		ProviderPaymentID: providerPaymentID, PaymentID: "unknown", Status: repository.GatewayPaymentApproved,
	}, nil)
	environment.paymentRepo.On("FindByID", mock.Anything, "unknown").Return(nil, repository.ErrPaymentNotFound)

	recorder := signedWebhook(environment, testWebhookSecret, paymentNotice)

	assertStatus(t, http.StatusOK, recorder)
	assert.JSONEq(t, `{"status":"ignored"}`, recorder.Body.String())
}

func TestMercadoPagoWebhook_ShouldAskForRetryWhenGatewayIsUnavailable(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayMercadoPago)
	environment.gateway.On("FindPayment", mock.Anything, providerPaymentID).Return(repository.GatewayPayment{}, repository.ErrPaymentGatewayUnavailable)

	recorder := signedWebhook(environment, testWebhookSecret, paymentNotice)

	assertStatus(t, http.StatusBadGateway, recorder)
}

func TestMercadoPagoWebhook_ShouldRejectInvalidBody(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayMercadoPago)

	recorder := signedWebhook(environment, testWebhookSecret, `not json`)

	assertStatus(t, http.StatusBadRequest, recorder)
}

func TestMercadoPagoWebhook_ShouldNotExistWithFakeGateway(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)

	recorder := signedWebhook(environment, testWebhookSecret, paymentNotice)

	assertStatus(t, http.StatusNotFound, recorder)
}
