package model

import (
	"time"

	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/valueobject"
)

type PaymentModel struct {
	ID                string `gorm:"primaryKey;type:uuid"`
	BudgetID          string `gorm:"type:uuid"`
	OrderID           string `gorm:"type:uuid"`
	SagaID            string `gorm:"type:uuid"`
	Amount            float64
	Status            string
	Provider          string
	CheckoutReference string
	CheckoutURL       string
	ProviderPaymentID string
	FailureReason     string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (PaymentModel) TableName() string { return "payments" }

func PaymentModelFromDomain(payment *entity.Payment) *PaymentModel {
	return &PaymentModel{
		ID:                payment.ID(),
		BudgetID:          payment.BudgetID(),
		OrderID:           payment.OrderID(),
		SagaID:            payment.SagaID(),
		Amount:            payment.Amount(),
		Status:            payment.Status().String(),
		Provider:          payment.Provider(),
		CheckoutReference: payment.CheckoutReference(),
		CheckoutURL:       payment.CheckoutURL(),
		ProviderPaymentID: payment.ProviderPaymentID(),
		FailureReason:     payment.FailureReason(),
		CreatedAt:         payment.CreatedAt(),
		UpdatedAt:         payment.UpdatedAt(),
	}
}

func (m *PaymentModel) ToDomain() *entity.Payment {
	return entity.RestorePayment(entity.PaymentSnapshot{
		ID:                m.ID,
		BudgetID:          m.BudgetID,
		OrderID:           m.OrderID,
		SagaID:            m.SagaID,
		Amount:            m.Amount,
		Status:            valueobject.PaymentStatus(m.Status),
		Provider:          m.Provider,
		CheckoutReference: m.CheckoutReference,
		CheckoutURL:       m.CheckoutURL,
		ProviderPaymentID: m.ProviderPaymentID,
		FailureReason:     m.FailureReason,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	})
}
