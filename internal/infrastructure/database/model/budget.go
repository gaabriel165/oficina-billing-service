package model

import (
	"time"

	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/valueobject"
)

type BudgetItemModel struct {
	ID          string `gorm:"primaryKey;type:uuid"`
	BudgetID    string `gorm:"type:uuid"`
	Kind        string
	ReferenceID string
	Description string
	Quantity    int
	UnitPrice   float64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (BudgetItemModel) TableName() string { return "budget_items" }

type BudgetModel struct {
	ID              string `gorm:"primaryKey;type:uuid"`
	OrderID         string `gorm:"type:uuid"`
	SagaID          string `gorm:"type:uuid"`
	CustomerID      string `gorm:"type:uuid"`
	CustomerName    string
	CustomerEmail   string
	Status          string
	TotalAmount     float64
	RejectionReason string
	DecidedAt       *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Items           []BudgetItemModel `gorm:"foreignKey:BudgetID"`
}

func (BudgetModel) TableName() string { return "budgets" }

func BudgetModelFromDomain(budget *entity.Budget) *BudgetModel {
	items := make([]BudgetItemModel, 0, len(budget.Items()))
	for _, item := range budget.Items() {
		items = append(items, BudgetItemModel{
			ID:          item.ID(),
			BudgetID:    budget.ID(),
			Kind:        item.Kind().String(),
			ReferenceID: item.ReferenceID(),
			Description: item.Description(),
			Quantity:    item.Quantity(),
			UnitPrice:   item.UnitPrice(),
			CreatedAt:   budget.CreatedAt(),
			UpdatedAt:   budget.UpdatedAt(),
		})
	}

	return &BudgetModel{
		ID:              budget.ID(),
		OrderID:         budget.OrderID(),
		SagaID:          budget.SagaID(),
		CustomerID:      budget.Customer().ID,
		CustomerName:    budget.Customer().Name,
		CustomerEmail:   budget.Customer().Email,
		Status:          budget.Status().String(),
		TotalAmount:     budget.TotalAmount(),
		RejectionReason: budget.RejectionReason(),
		DecidedAt:       budget.DecidedAt(),
		CreatedAt:       budget.CreatedAt(),
		UpdatedAt:       budget.UpdatedAt(),
		Items:           items,
	}
}

func (m *BudgetModel) ToDomain() *entity.Budget {
	items := make([]*entity.BudgetItem, 0, len(m.Items))
	for _, item := range m.Items {
		items = append(items, entity.RestoreBudgetItem(
			item.ID, valueobject.BudgetItemKind(item.Kind), item.ReferenceID, item.Description, item.Quantity, item.UnitPrice,
		))
	}

	return entity.RestoreBudget(entity.BudgetSnapshot{
		ID:              m.ID,
		OrderID:         m.OrderID,
		SagaID:          m.SagaID,
		Customer:        entity.BudgetCustomer{ID: m.CustomerID, Name: m.CustomerName, Email: m.CustomerEmail},
		Status:          valueobject.BudgetStatus(m.Status),
		Items:           items,
		TotalAmount:     m.TotalAmount,
		RejectionReason: m.RejectionReason,
		DecidedAt:       m.DecidedAt,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	})
}
