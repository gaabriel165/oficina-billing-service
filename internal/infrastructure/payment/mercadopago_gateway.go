package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
)

const (
	ProviderMercadoPago = "mercadopago"

	MercadoPagoWebhookPath = "/webhooks/mercadopago"

	currencyBRL             = "BRL"
	statusApproved          = "approved"
	statusRejected          = "rejected"
	statusCancelled         = "cancelled"
	headerIdempotencyKey    = "X-Idempotency-Key"
	refundIdempotencyPrefix = "refund-"
)

type MercadoPagoGateway struct {
	client          *http.Client
	baseURL         string
	accessToken     string
	notificationURL string
}

func NewMercadoPagoGateway(client *http.Client, baseURL, accessToken, publicBaseURL string) *MercadoPagoGateway {
	return &MercadoPagoGateway{
		client:          client,
		baseURL:         strings.TrimRight(baseURL, "/"),
		accessToken:     accessToken,
		notificationURL: strings.TrimRight(publicBaseURL, "/") + MercadoPagoWebhookPath,
	}
}

type preferenceItem struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Quantity   int     `json:"quantity"`
	UnitPrice  float64 `json:"unit_price"`
	CurrencyID string  `json:"currency_id"`
}

type preferencePayer struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

type preferenceRequest struct {
	Items             []preferenceItem `json:"items"`
	Payer             preferencePayer  `json:"payer"`
	ExternalReference string           `json:"external_reference"`
	NotificationURL   string           `json:"notification_url"`
}

type preferenceResponse struct {
	ID               string `json:"id"`
	InitPoint        string `json:"init_point"`
	SandboxInitPoint string `json:"sandbox_init_point"`
}

type paymentResponse struct {
	ID                int64  `json:"id"`
	Status            string `json:"status"`
	StatusDetail      string `json:"status_detail"`
	ExternalReference string `json:"external_reference"`
}

func (g *MercadoPagoGateway) Provider() string {
	return ProviderMercadoPago
}

func (g *MercadoPagoGateway) CreateCheckout(ctx context.Context, payment *entity.Payment, budget *entity.Budget) (repository.Checkout, error) {
	request := preferenceRequest{
		Items:             preferenceItems(budget),
		Payer:             preferencePayer{Name: budget.Customer().Name, Email: budget.Customer().Email},
		ExternalReference: payment.ID(),
		NotificationURL:   g.notificationURL,
	}

	var preference preferenceResponse
	if err := g.send(ctx, http.MethodPost, "/checkout/preferences", "", request, &preference); err != nil {
		return repository.Checkout{}, err
	}

	return repository.Checkout{Reference: preference.ID, URL: checkoutURL(preference)}, nil
}

func (g *MercadoPagoGateway) FindPayment(ctx context.Context, providerPaymentID string) (repository.GatewayPayment, error) {
	var found paymentResponse
	if err := g.send(ctx, http.MethodGet, "/v1/payments/"+providerPaymentID, "", nil, &found); err != nil {
		return repository.GatewayPayment{}, err
	}

	return repository.GatewayPayment{
		ProviderPaymentID: strconv.FormatInt(found.ID, 10),
		PaymentID:         found.ExternalReference,
		Status:            gatewayStatus(found.Status),
		StatusDetail:      found.StatusDetail,
	}, nil
}

func (g *MercadoPagoGateway) Refund(ctx context.Context, payment *entity.Payment) error {
	path := "/v1/payments/" + payment.ProviderPaymentID() + "/refunds"
	return g.send(ctx, http.MethodPost, path, refundIdempotencyPrefix+payment.ID(), struct{}{}, nil)
}

func (g *MercadoPagoGateway) send(ctx context.Context, method, path, idempotencyKey string, body, target any) error {
	request, err := g.newRequest(ctx, method, path, body)
	if err != nil {
		return err
	}
	if idempotencyKey != "" {
		request.Header.Set(headerIdempotencyKey, idempotencyKey)
	}

	response, err := g.client.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %s", repository.ErrPaymentGatewayUnavailable, err.Error())
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%w: %s %s answered %d", repository.ErrPaymentGatewayUnavailable, method, path, response.StatusCode)
	}
	if target == nil {
		return nil
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("%w: invalid response body", repository.ErrPaymentGatewayUnavailable)
	}
	return nil
}

func (g *MercadoPagoGateway) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		payload = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, g.baseURL+path, payload)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+g.accessToken)
	request.Header.Set("Content-Type", "application/json")
	return request, nil
}

func preferenceItems(budget *entity.Budget) []preferenceItem {
	items := make([]preferenceItem, 0, len(budget.Items()))
	for _, item := range budget.Items() {
		items = append(items, preferenceItem{
			ID:         item.ReferenceID(),
			Title:      item.Description(),
			Quantity:   item.Quantity(),
			UnitPrice:  item.UnitPrice(),
			CurrencyID: currencyBRL,
		})
	}
	return items
}

func checkoutURL(preference preferenceResponse) string {
	if preference.InitPoint != "" {
		return preference.InitPoint
	}
	return preference.SandboxInitPoint
}

func gatewayStatus(status string) repository.GatewayPaymentStatus {
	switch status {
	case statusApproved:
		return repository.GatewayPaymentApproved
	case statusRejected, statusCancelled:
		return repository.GatewayPaymentRejected
	default:
		return repository.GatewayPaymentPending
	}
}
