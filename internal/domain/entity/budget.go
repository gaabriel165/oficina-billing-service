package entity

import (
	"time"

	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/valueobject"
	"github.com/google/uuid"
)

type BudgetCustomer struct {
	ID    string
	Name  string
	Email string
}

type Budget struct {
	id              string
	orderID         string
	sagaID          string
	customer        BudgetCustomer
	status          valueobject.BudgetStatus
	items           []*BudgetItem
	totalAmount     float64
	rejectionReason string
	decidedAt       *time.Time
	createdAt       time.Time
	updatedAt       time.Time
}

type BudgetSnapshot struct {
	ID              string
	OrderID         string
	SagaID          string
	Customer        BudgetCustomer
	Status          valueobject.BudgetStatus
	Items           []*BudgetItem
	TotalAmount     float64
	RejectionReason string
	DecidedAt       *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func NewBudget(orderID, sagaID string, customer BudgetCustomer, items []*BudgetItem) (*Budget, error) {
	if orderID == "" {
		return nil, ErrBudgetOrderRequired
	}
	if sagaID == "" {
		return nil, ErrBudgetSagaRequired
	}
	if customer.ID == "" {
		return nil, ErrBudgetCustomerRequired
	}
	if len(items) == 0 {
		return nil, ErrBudgetNoItems
	}

	now := time.Now()
	return &Budget{
		id:          uuid.NewString(),
		orderID:     orderID,
		sagaID:      sagaID,
		customer:    customer,
		status:      valueobject.BudgetStatusPendingApproval,
		items:       items,
		totalAmount: sumItems(items),
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

func RestoreBudget(snapshot BudgetSnapshot) *Budget {
	return &Budget{
		id:              snapshot.ID,
		orderID:         snapshot.OrderID,
		sagaID:          snapshot.SagaID,
		customer:        snapshot.Customer,
		status:          snapshot.Status,
		items:           snapshot.Items,
		totalAmount:     snapshot.TotalAmount,
		rejectionReason: snapshot.RejectionReason,
		decidedAt:       snapshot.DecidedAt,
		createdAt:       snapshot.CreatedAt,
		updatedAt:       snapshot.UpdatedAt,
	}
}

func (b *Budget) ID() string                       { return b.id }
func (b *Budget) OrderID() string                  { return b.orderID }
func (b *Budget) SagaID() string                   { return b.sagaID }
func (b *Budget) Customer() BudgetCustomer         { return b.customer }
func (b *Budget) Status() valueobject.BudgetStatus { return b.status }
func (b *Budget) Items() []*BudgetItem             { return b.items }
func (b *Budget) TotalAmount() float64             { return b.totalAmount }
func (b *Budget) RejectionReason() string          { return b.rejectionReason }
func (b *Budget) DecidedAt() *time.Time            { return b.decidedAt }
func (b *Budget) CreatedAt() time.Time             { return b.createdAt }
func (b *Budget) UpdatedAt() time.Time             { return b.updatedAt }

func (b *Budget) IsApproved() bool {
	return b.status == valueobject.BudgetStatusApproved
}

func (b *Budget) IsCancelled() bool {
	return b.status == valueobject.BudgetStatusCancelled
}

func (b *Budget) BelongsTo(customerID string) bool {
	return b.customer.ID == customerID
}

func (b *Budget) Approve() error {
	return b.decide(valueobject.BudgetStatusApproved)
}

func (b *Budget) Reject(reason string) error {
	if reason == "" {
		return ErrBudgetReasonRequired
	}
	if err := b.decide(valueobject.BudgetStatusRejected); err != nil {
		return err
	}
	b.rejectionReason = reason
	return nil
}

func (b *Budget) Cancel() error {
	return b.transitionTo(valueobject.BudgetStatusCancelled)
}

func (b *Budget) decide(next valueobject.BudgetStatus) error {
	if err := b.transitionTo(next); err != nil {
		return err
	}
	now := time.Now()
	b.decidedAt = &now
	return nil
}

func (b *Budget) transitionTo(next valueobject.BudgetStatus) error {
	if !b.status.CanTransitionTo(next) {
		return ErrBudgetInvalidTransition
	}
	b.status = next
	b.updatedAt = time.Now()
	return nil
}

func sumItems(items []*BudgetItem) float64 {
	total := 0.0
	for _, item := range items {
		total += item.TotalPrice()
	}
	return total
}
