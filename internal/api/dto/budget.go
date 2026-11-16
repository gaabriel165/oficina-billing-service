package dto

import (
	"time"

	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
)

type RejectBudgetRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type BudgetItemResponse struct {
	ID          string  `json:"id"`
	Kind        string  `json:"kind"`
	ReferenceID string  `json:"reference_id"`
	Description string  `json:"description"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	TotalPrice  float64 `json:"total_price"`
}

type BudgetResponse struct {
	ID              string               `json:"id"`
	OrderID         string               `json:"order_id"`
	SagaID          string               `json:"saga_id"`
	CustomerID      string               `json:"customer_id"`
	Status          string               `json:"status"`
	Items           []BudgetItemResponse `json:"items"`
	TotalAmount     float64              `json:"total_amount"`
	RejectionReason string               `json:"rejection_reason,omitempty"`
	DecidedAt       *time.Time           `json:"decided_at,omitempty"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
}

func ToBudgetResponse(budget *entity.Budget) BudgetResponse {
	items := make([]BudgetItemResponse, 0, len(budget.Items()))
	for _, item := range budget.Items() {
		items = append(items, BudgetItemResponse{
			ID:          item.ID(),
			Kind:        item.Kind().String(),
			ReferenceID: item.ReferenceID(),
			Description: item.Description(),
			Quantity:    item.Quantity(),
			UnitPrice:   item.UnitPrice(),
			TotalPrice:  item.TotalPrice(),
		})
	}

	return BudgetResponse{
		ID:              budget.ID(),
		OrderID:         budget.OrderID(),
		SagaID:          budget.SagaID(),
		CustomerID:      budget.Customer().ID,
		Status:          budget.Status().String(),
		Items:           items,
		TotalAmount:     budget.TotalAmount(),
		RejectionReason: budget.RejectionReason(),
		DecidedAt:       budget.DecidedAt(),
		CreatedAt:       budget.CreatedAt(),
		UpdatedAt:       budget.UpdatedAt(),
	}
}
