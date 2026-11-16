package entity

import (
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/valueobject"
	"github.com/google/uuid"
)

type BudgetItem struct {
	id          string
	kind        valueobject.BudgetItemKind
	referenceID string
	description string
	quantity    int
	unitPrice   float64
}

func NewBudgetItem(kind valueobject.BudgetItemKind, referenceID, description string, quantity int, unitPrice float64) (*BudgetItem, error) {
	if description == "" {
		return nil, ErrBudgetItemDescription
	}
	if quantity <= 0 {
		return nil, ErrBudgetItemInvalidQty
	}
	if unitPrice <= 0 {
		return nil, ErrBudgetItemInvalidPrice
	}

	return &BudgetItem{
		id:          uuid.NewString(),
		kind:        kind,
		referenceID: referenceID,
		description: description,
		quantity:    quantity,
		unitPrice:   unitPrice,
	}, nil
}

func RestoreBudgetItem(id string, kind valueobject.BudgetItemKind, referenceID, description string, quantity int, unitPrice float64) *BudgetItem {
	return &BudgetItem{id: id, kind: kind, referenceID: referenceID, description: description, quantity: quantity, unitPrice: unitPrice}
}

func (i *BudgetItem) ID() string                       { return i.id }
func (i *BudgetItem) Kind() valueobject.BudgetItemKind { return i.kind }
func (i *BudgetItem) ReferenceID() string              { return i.referenceID }
func (i *BudgetItem) Description() string              { return i.description }
func (i *BudgetItem) Quantity() int                    { return i.quantity }
func (i *BudgetItem) UnitPrice() float64               { return i.unitPrice }
func (i *BudgetItem) TotalPrice() float64              { return i.unitPrice * float64(i.quantity) }
