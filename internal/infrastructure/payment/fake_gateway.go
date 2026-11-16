package payment

import (
	"context"
	"fmt"
	"strings"

	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
)

const ProviderFake = "fake"

type FakeGateway struct {
	publicBaseURL string
}

func NewFakeGateway(publicBaseURL string) *FakeGateway {
	return &FakeGateway{publicBaseURL: strings.TrimRight(publicBaseURL, "/")}
}

func (g *FakeGateway) Provider() string {
	return ProviderFake
}

func (g *FakeGateway) CreateCheckout(_ context.Context, payment *entity.Payment, _ *entity.Budget) (repository.Checkout, error) {
	return repository.Checkout{
		Reference: "fake-" + payment.ID(),
		URL:       fmt.Sprintf("%s/api/v1/payments/%s/simulate", g.publicBaseURL, payment.ID()),
	}, nil
}

func (g *FakeGateway) FindPayment(_ context.Context, _ string) (repository.GatewayPayment, error) {
	return repository.GatewayPayment{}, repository.ErrPaymentGatewayUnavailable
}

func (g *FakeGateway) Refund(_ context.Context, _ *entity.Payment) error {
	return nil
}
