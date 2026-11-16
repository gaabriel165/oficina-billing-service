package entity

import (
	"time"

	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/valueobject"
	"github.com/google/uuid"
)

type Payment struct {
	id                string
	budgetID          string
	orderID           string
	sagaID            string
	amount            float64
	status            valueobject.PaymentStatus
	provider          string
	checkoutReference string
	checkoutURL       string
	providerPaymentID string
	failureReason     string
	createdAt         time.Time
	updatedAt         time.Time
}

type PaymentSnapshot struct {
	ID                string
	BudgetID          string
	OrderID           string
	SagaID            string
	Amount            float64
	Status            valueobject.PaymentStatus
	Provider          string
	CheckoutReference string
	CheckoutURL       string
	ProviderPaymentID string
	FailureReason     string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func NewPayment(budget *Budget, provider string) (*Payment, error) {
	if budget == nil {
		return nil, ErrPaymentBudgetRequired
	}
	if !budget.IsApproved() {
		return nil, ErrBudgetNotApproved
	}
	if budget.TotalAmount() <= 0 {
		return nil, ErrPaymentInvalidAmount
	}

	now := time.Now()
	return &Payment{
		id:        uuid.NewString(),
		budgetID:  budget.ID(),
		orderID:   budget.OrderID(),
		sagaID:    budget.SagaID(),
		amount:    budget.TotalAmount(),
		status:    valueobject.PaymentStatusPending,
		provider:  provider,
		createdAt: now,
		updatedAt: now,
	}, nil
}

func RestorePayment(snapshot PaymentSnapshot) *Payment {
	return &Payment{
		id:                snapshot.ID,
		budgetID:          snapshot.BudgetID,
		orderID:           snapshot.OrderID,
		sagaID:            snapshot.SagaID,
		amount:            snapshot.Amount,
		status:            snapshot.Status,
		provider:          snapshot.Provider,
		checkoutReference: snapshot.CheckoutReference,
		checkoutURL:       snapshot.CheckoutURL,
		providerPaymentID: snapshot.ProviderPaymentID,
		failureReason:     snapshot.FailureReason,
		createdAt:         snapshot.CreatedAt,
		updatedAt:         snapshot.UpdatedAt,
	}
}

func (p *Payment) ID() string                        { return p.id }
func (p *Payment) BudgetID() string                  { return p.budgetID }
func (p *Payment) OrderID() string                   { return p.orderID }
func (p *Payment) SagaID() string                    { return p.sagaID }
func (p *Payment) Amount() float64                   { return p.amount }
func (p *Payment) Status() valueobject.PaymentStatus { return p.status }
func (p *Payment) Provider() string                  { return p.provider }
func (p *Payment) CheckoutReference() string         { return p.checkoutReference }
func (p *Payment) CheckoutURL() string               { return p.checkoutURL }
func (p *Payment) ProviderPaymentID() string         { return p.providerPaymentID }
func (p *Payment) FailureReason() string             { return p.failureReason }
func (p *Payment) CreatedAt() time.Time              { return p.createdAt }
func (p *Payment) UpdatedAt() time.Time              { return p.updatedAt }

func (p *Payment) IsPending() bool {
	return p.status == valueobject.PaymentStatusPending
}

func (p *Payment) IsConfirmed() bool {
	return p.status == valueobject.PaymentStatusConfirmed
}

func (p *Payment) AttachCheckout(reference, url string) error {
	if reference == "" {
		return ErrPaymentCheckoutRequired
	}
	p.checkoutReference = reference
	p.checkoutURL = url
	p.updatedAt = time.Now()
	return nil
}

func (p *Payment) Confirm(providerPaymentID string) error {
	if err := p.transitionTo(valueobject.PaymentStatusConfirmed); err != nil {
		return err
	}
	p.providerPaymentID = providerPaymentID
	return nil
}

func (p *Payment) Fail(providerPaymentID, reason string) error {
	if err := p.transitionTo(valueobject.PaymentStatusFailed); err != nil {
		return err
	}
	p.providerPaymentID = providerPaymentID
	p.failureReason = reason
	return nil
}

func (p *Payment) Refund() error {
	return p.transitionTo(valueobject.PaymentStatusRefunded)
}

func (p *Payment) transitionTo(next valueobject.PaymentStatus) error {
	if !p.status.CanTransitionTo(next) {
		return ErrPaymentInvalidTransition
	}
	p.status = next
	p.updatedAt = time.Now()
	return nil
}
