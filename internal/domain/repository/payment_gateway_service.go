package repository

import (
	"context"
	"errors"

	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
)

var ErrPaymentGatewayUnavailable = errors.New("payment gateway unavailable")

type Checkout struct {
	Reference string
	URL       string
}

type GatewayPaymentStatus string

const (
	GatewayPaymentApproved GatewayPaymentStatus = "approved"
	GatewayPaymentRejected GatewayPaymentStatus = "rejected"
	GatewayPaymentPending  GatewayPaymentStatus = "pending"
)

type GatewayPayment struct {
	ProviderPaymentID string
	PaymentID         string
	Status            GatewayPaymentStatus
	StatusDetail      string
}

type PaymentGatewayService interface {
	Provider() string
	CreateCheckout(ctx context.Context, payment *entity.Payment, budget *entity.Budget) (Checkout, error)
	FindPayment(ctx context.Context, providerPaymentID string) (GatewayPayment, error)
	Refund(ctx context.Context, payment *entity.Payment) error
}
