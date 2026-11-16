package api_test

import (
	"net/http"
	"testing"

	"github.com/gabrielcamargo/oficina-billing-service/configs"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetPayment_ShouldReturnPaymentToOperator(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)
	payment := pendingPayment(t)
	environment.paymentRepo.On("FindByID", mock.Anything, payment.ID()).Return(payment, nil)

	recorder := environment.request(t, http.MethodGet, "/api/v1/payments/"+payment.ID(), roleOperator, "operator-1", "")

	assertStatus(t, http.StatusOK, recorder)
	assert.Contains(t, recorder.Body.String(), `"status":"pending"`)
}

func TestGetPayment_ShouldForbidCustomer(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)

	recorder := environment.request(t, http.MethodGet, "/api/v1/payments/payment-1", roleCustomer, "customer-1", "")

	assertStatus(t, http.StatusForbidden, recorder)
}

func TestGetPayment_ShouldReturnNotFound(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)
	environment.paymentRepo.On("FindByID", mock.Anything, "missing").Return(nil, repository.ErrPaymentNotFound)

	recorder := environment.request(t, http.MethodGet, "/api/v1/payments/missing", roleOperator, "operator-1", "")

	assertStatus(t, http.StatusNotFound, recorder)
}

func TestSimulatePayment_ShouldConfirmPayment(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)
	payment := pendingPayment(t)
	environment.paymentRepo.On("FindByID", mock.Anything, payment.ID()).Return(payment, nil)
	environment.paymentRepo.On("Update", mock.Anything, payment, replyOfType(messaging.TypePaymentConfirmed)).Return(nil)

	recorder := environment.request(t, http.MethodPost, "/api/v1/payments/"+payment.ID()+"/simulate", roleCustomer, "customer-1", `{"approved":true}`)

	assertStatus(t, http.StatusOK, recorder)
	assert.Contains(t, recorder.Body.String(), `"status":"confirmed"`)
	assert.Contains(t, recorder.Body.String(), `"provider_payment_id":"simulated-`+payment.ID()+`"`)
}

func TestSimulatePayment_ShouldFailPaymentWithReason(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)
	payment := pendingPayment(t)
	environment.paymentRepo.On("FindByID", mock.Anything, payment.ID()).Return(payment, nil)
	environment.paymentRepo.On("Update", mock.Anything, payment, replyOfType(messaging.TypePaymentFailed)).Return(nil)

	recorder := environment.request(t, http.MethodPost, "/api/v1/payments/"+payment.ID()+"/simulate", roleCustomer, "customer-1", `{"approved":false,"reason":"card declined"}`)

	assertStatus(t, http.StatusOK, recorder)
	assert.Contains(t, recorder.Body.String(), `"failure_reason":"card declined"`)
}

func TestSimulatePayment_ShouldReturnBadRequestForInvalidBody(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)

	recorder := environment.request(t, http.MethodPost, "/api/v1/payments/payment-1/simulate", roleCustomer, "customer-1", `{"approved":"yes"}`)

	assertStatus(t, http.StatusBadRequest, recorder)
}

func TestSimulatePayment_ShouldReturnNotFoundWhenPaymentDoesNotExist(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)
	environment.paymentRepo.On("FindByID", mock.Anything, "missing").Return(nil, repository.ErrPaymentNotFound)

	recorder := environment.request(t, http.MethodPost, "/api/v1/payments/missing/simulate", roleCustomer, "customer-1", `{"approved":true}`)

	assertStatus(t, http.StatusNotFound, recorder)
}

func TestSimulatePayment_ShouldNotExistWithRealGateway(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayMercadoPago)

	recorder := environment.request(t, http.MethodPost, "/api/v1/payments/payment-1/simulate", roleCustomer, "customer-1", `{"approved":true}`)

	assertStatus(t, http.StatusNotFound, recorder)
}
