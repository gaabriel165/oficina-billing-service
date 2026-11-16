package dto

import (
	"time"

	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
)

type SimulatePaymentRequest struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason"`
}

type PaymentResponse struct {
	ID                string    `json:"id"`
	BudgetID          string    `json:"budget_id"`
	OrderID           string    `json:"order_id"`
	Amount            float64   `json:"amount"`
	Status            string    `json:"status"`
	Provider          string    `json:"provider"`
	CheckoutURL       string    `json:"checkout_url"`
	ProviderPaymentID string    `json:"provider_payment_id,omitempty"`
	FailureReason     string    `json:"failure_reason,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func ToPaymentResponse(payment *entity.Payment) PaymentResponse {
	return PaymentResponse{
		ID:                payment.ID(),
		BudgetID:          payment.BudgetID(),
		OrderID:           payment.OrderID(),
		Amount:            payment.Amount(),
		Status:            payment.Status().String(),
		Provider:          payment.Provider(),
		CheckoutURL:       payment.CheckoutURL(),
		ProviderPaymentID: payment.ProviderPaymentID(),
		FailureReason:     payment.FailureReason(),
		CreatedAt:         payment.CreatedAt(),
		UpdatedAt:         payment.UpdatedAt(),
	}
}
