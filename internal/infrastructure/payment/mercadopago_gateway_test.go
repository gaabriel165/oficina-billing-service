package payment_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/valueobject"
	"github.com/gabrielcamargo/oficina-billing-service/internal/infrastructure/payment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordedRequest struct {
	method         string
	path           string
	authorization  string
	idempotencyKey string
	body           map[string]any
}

type mercadoPagoStub struct {
	server   *httptest.Server
	requests []recordedRequest
	status   int
	response string
}

func newMercadoPagoStub(t *testing.T, status int, response string) *mercadoPagoStub {
	t.Helper()
	stub := &mercadoPagoStub{status: status, response: response}
	stub.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		recorded := recordedRequest{
			method:         r.Method,
			path:           r.URL.Path,
			authorization:  r.Header.Get("Authorization"),
			idempotencyKey: r.Header.Get("X-Idempotency-Key"),
		}
		if len(raw) > 0 {
			require.NoError(t, json.Unmarshal(raw, &recorded.body))
		}
		stub.requests = append(stub.requests, recorded)
		w.WriteHeader(stub.status)
		_, err = w.Write([]byte(stub.response))
		require.NoError(t, err)
	}))
	t.Cleanup(stub.server.Close)
	return stub
}

func (s *mercadoPagoStub) gateway() *payment.MercadoPagoGateway {
	return payment.NewMercadoPagoGateway(s.server.Client(), s.server.URL+"/", "TEST-token", "https://api.oficina.example/")
}

func approvedBudget(t *testing.T) *entity.Budget {
	t.Helper()
	service, err := entity.NewBudgetItem(valueobject.BudgetItemKindService, "service-1", "Troca de óleo", 1, 120)
	require.NoError(t, err)
	part, err := entity.NewBudgetItem(valueobject.BudgetItemKindPart, "part-1", "Filtro de óleo", 2, 29.9)
	require.NoError(t, err)
	budget, err := entity.NewBudget("order-1", "saga-1",
		entity.BudgetCustomer{ID: "customer-1", Name: "Maria Silva", Email: "maria@example.com"},
		[]*entity.BudgetItem{service, part})
	require.NoError(t, err)
	require.NoError(t, budget.Approve())
	return budget
}

func pendingPayment(t *testing.T, budget *entity.Budget) *entity.Payment {
	t.Helper()
	pending, err := entity.NewPayment(budget, payment.ProviderMercadoPago)
	require.NoError(t, err)
	return pending
}

func TestMercadoPagoGateway_ShouldCreatePreferenceWithBudgetItems(t *testing.T) {
	stub := newMercadoPagoStub(t, http.StatusCreated, `{"id":"pref-123","init_point":"https://www.mercadopago.com.br/checkout/v1/redirect?pref_id=pref-123","sandbox_init_point":"https://sandbox.mercadopago.com.br/checkout/v1/redirect?pref_id=pref-123"}`)
	budget := approvedBudget(t)
	pending := pendingPayment(t, budget)

	checkout, err := stub.gateway().CreateCheckout(context.Background(), pending, budget)

	require.NoError(t, err)
	assert.Equal(t, "pref-123", checkout.Reference)
	assert.Equal(t, "https://www.mercadopago.com.br/checkout/v1/redirect?pref_id=pref-123", checkout.URL)
	require.Len(t, stub.requests, 1)
	request := stub.requests[0]
	assert.Equal(t, http.MethodPost, request.method)
	assert.Equal(t, "/checkout/preferences", request.path)
	assert.Equal(t, "Bearer TEST-token", request.authorization)
	assert.Equal(t, pending.ID(), request.body["external_reference"])
	assert.Equal(t, "https://api.oficina.example/api/v1/webhooks/mercadopago", request.body["notification_url"])
	assert.Equal(t, map[string]any{"name": "Maria Silva", "email": "maria@example.com"}, request.body["payer"])
	items, ok := request.body["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 2)
	assert.Equal(t, map[string]any{
		"id": "part-1", "title": "Filtro de óleo", "quantity": 2.0, "unit_price": 29.9, "currency_id": "BRL",
	}, items[1])
}

func TestMercadoPagoGateway_ShouldFallBackToSandboxCheckoutLink(t *testing.T) {
	stub := newMercadoPagoStub(t, http.StatusCreated, `{"id":"pref-123","sandbox_init_point":"https://sandbox.mercadopago.com.br/checkout"}`)
	budget := approvedBudget(t)

	checkout, err := stub.gateway().CreateCheckout(context.Background(), pendingPayment(t, budget), budget)

	require.NoError(t, err)
	assert.Equal(t, "https://sandbox.mercadopago.com.br/checkout", checkout.URL)
}

func TestMercadoPagoGateway_ShouldReportUnavailableWhenPreferenceIsRefused(t *testing.T) {
	stub := newMercadoPagoStub(t, http.StatusUnauthorized, `{"message":"invalid access token"}`)
	budget := approvedBudget(t)

	_, err := stub.gateway().CreateCheckout(context.Background(), pendingPayment(t, budget), budget)

	assert.ErrorIs(t, err, repository.ErrPaymentGatewayUnavailable)
}

func TestMercadoPagoGateway_ShouldReportUnavailableWhenResponseIsNotJSON(t *testing.T) {
	stub := newMercadoPagoStub(t, http.StatusOK, `<html>`)
	budget := approvedBudget(t)

	_, err := stub.gateway().CreateCheckout(context.Background(), pendingPayment(t, budget), budget)

	assert.ErrorIs(t, err, repository.ErrPaymentGatewayUnavailable)
}

func TestMercadoPagoGateway_ShouldReportUnavailableWhenServerIsDown(t *testing.T) {
	stub := newMercadoPagoStub(t, http.StatusOK, `{}`)
	gateway := stub.gateway()
	stub.server.Close()

	_, err := gateway.FindPayment(context.Background(), "999")

	assert.ErrorIs(t, err, repository.ErrPaymentGatewayUnavailable)
}

func TestMercadoPagoGateway_ShouldMapPaymentStatus(t *testing.T) {
	cases := map[string]repository.GatewayPaymentStatus{
		"approved":   repository.GatewayPaymentApproved,
		"rejected":   repository.GatewayPaymentRejected,
		"cancelled":  repository.GatewayPaymentRejected,
		"pending":    repository.GatewayPaymentPending,
		"in_process": repository.GatewayPaymentPending,
	}

	for status, expected := range cases {
		t.Run(status, func(t *testing.T) {
			stub := newMercadoPagoStub(t, http.StatusOK, `{"id":999999999,"status":"`+status+`","status_detail":"detail","external_reference":"payment-1"}`)

			found, err := stub.gateway().FindPayment(context.Background(), "999999999")

			require.NoError(t, err)
			assert.Equal(t, expected, found.Status)
			assert.Equal(t, "999999999", found.ProviderPaymentID)
			assert.Equal(t, "payment-1", found.PaymentID)
			assert.Equal(t, "detail", found.StatusDetail)
			assert.Equal(t, "/v1/payments/999999999", stub.requests[0].path)
			assert.Equal(t, http.MethodGet, stub.requests[0].method)
		})
	}
}

func TestMercadoPagoGateway_ShouldRefundWithIdempotencyKey(t *testing.T) {
	stub := newMercadoPagoStub(t, http.StatusCreated, `{"id":1,"status":"approved"}`)
	confirmed := entity.RestorePayment(entity.PaymentSnapshot{
		ID: "payment-1", Status: valueobject.PaymentStatusConfirmed, ProviderPaymentID: "999999999",
	})

	err := stub.gateway().Refund(context.Background(), confirmed)

	require.NoError(t, err)
	require.Len(t, stub.requests, 1)
	assert.Equal(t, http.MethodPost, stub.requests[0].method)
	assert.Equal(t, "/v1/payments/999999999/refunds", stub.requests[0].path)
	assert.Equal(t, "refund-payment-1", stub.requests[0].idempotencyKey)
}

func TestMercadoPagoGateway_ShouldReportUnavailableWhenRefundIsRefused(t *testing.T) {
	stub := newMercadoPagoStub(t, http.StatusBadRequest, `{"message":"payment already refunded"}`)
	confirmed := entity.RestorePayment(entity.PaymentSnapshot{ID: "payment-1", ProviderPaymentID: "999999999"})

	err := stub.gateway().Refund(context.Background(), confirmed)

	assert.ErrorIs(t, err, repository.ErrPaymentGatewayUnavailable)
}

func TestMercadoPagoGateway_ShouldIdentifyItself(t *testing.T) {
	assert.Equal(t, "mercadopago", payment.NewMercadoPagoGateway(http.DefaultClient, "", "", "").Provider())
}
