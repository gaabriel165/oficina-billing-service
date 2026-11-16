package entity_test

import (
	"testing"
	"time"

	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/valueobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var customer = entity.BudgetCustomer{ID: "customer-1", Name: "Maria Silva", Email: "maria@example.com"}

func newItems(t *testing.T) []*entity.BudgetItem {
	t.Helper()
	service, err := entity.NewBudgetItem(valueobject.BudgetItemKindService, "service-1", "Troca de óleo", 1, 120)
	require.NoError(t, err)
	part, err := entity.NewBudgetItem(valueobject.BudgetItemKindPart, "part-1", "Filtro de óleo", 2, 30)
	require.NoError(t, err)
	return []*entity.BudgetItem{service, part}
}

func newBudget(t *testing.T) *entity.Budget {
	t.Helper()
	budget, err := entity.NewBudget("order-1", "saga-1", customer, newItems(t))
	require.NoError(t, err)
	return budget
}

func TestNewBudgetItem_ShouldCreateItemWithTotalPrice(t *testing.T) {
	item, err := entity.NewBudgetItem(valueobject.BudgetItemKindPart, "part-1", "Filtro de óleo", 3, 29.9)

	require.NoError(t, err)
	assert.NotEmpty(t, item.ID())
	assert.Equal(t, valueobject.BudgetItemKindPart, item.Kind())
	assert.Equal(t, "part", item.Kind().String())
	assert.Equal(t, "part-1", item.ReferenceID())
	assert.Equal(t, "Filtro de óleo", item.Description())
	assert.Equal(t, 3, item.Quantity())
	assert.Equal(t, 29.9, item.UnitPrice())
	assert.InDelta(t, 89.7, item.TotalPrice(), 0.001)
}

func TestNewBudgetItem_ShouldRejectInvalidData(t *testing.T) {
	cases := []struct {
		name        string
		description string
		quantity    int
		unitPrice   float64
		expected    error
	}{
		{"empty description", "", 1, 10, entity.ErrBudgetItemDescription},
		{"zero quantity", "Filtro", 0, 10, entity.ErrBudgetItemInvalidQty},
		{"negative quantity", "Filtro", -1, 10, entity.ErrBudgetItemInvalidQty},
		{"zero price", "Filtro", 1, 0, entity.ErrBudgetItemInvalidPrice},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := entity.NewBudgetItem(valueobject.BudgetItemKindPart, "part-1", tc.description, tc.quantity, tc.unitPrice)

			assert.ErrorIs(t, err, tc.expected)
		})
	}
}

func TestRestoreBudgetItem_ShouldKeepGivenValues(t *testing.T) {
	item := entity.RestoreBudgetItem("item-1", valueobject.BudgetItemKindService, "service-1", "Alinhamento", 1, 80)

	assert.Equal(t, "item-1", item.ID())
	assert.Equal(t, valueobject.BudgetItemKindService, item.Kind())
	assert.Equal(t, 80.0, item.TotalPrice())
}

func TestNewBudget_ShouldStartPendingApprovalWithSummedTotal(t *testing.T) {
	budget := newBudget(t)

	assert.NotEmpty(t, budget.ID())
	assert.Equal(t, "order-1", budget.OrderID())
	assert.Equal(t, "saga-1", budget.SagaID())
	assert.Equal(t, customer, budget.Customer())
	assert.Equal(t, valueobject.BudgetStatusPendingApproval, budget.Status())
	assert.Len(t, budget.Items(), 2)
	assert.Equal(t, 180.0, budget.TotalAmount())
	assert.Empty(t, budget.RejectionReason())
	assert.Nil(t, budget.DecidedAt())
	assert.False(t, budget.CreatedAt().IsZero())
	assert.False(t, budget.UpdatedAt().IsZero())
	assert.True(t, budget.BelongsTo("customer-1"))
	assert.False(t, budget.BelongsTo("customer-2"))
}

func TestNewBudget_ShouldRejectMissingData(t *testing.T) {
	items := newItems(t)
	cases := []struct {
		name     string
		orderID  string
		sagaID   string
		customer entity.BudgetCustomer
		items    []*entity.BudgetItem
		expected error
	}{
		{"missing order", "", "saga-1", customer, items, entity.ErrBudgetOrderRequired},
		{"missing saga", "order-1", "", customer, items, entity.ErrBudgetSagaRequired},
		{"missing customer", "order-1", "saga-1", entity.BudgetCustomer{}, items, entity.ErrBudgetCustomerRequired},
		{"no items", "order-1", "saga-1", customer, nil, entity.ErrBudgetNoItems},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := entity.NewBudget(tc.orderID, tc.sagaID, tc.customer, tc.items)

			assert.ErrorIs(t, err, tc.expected)
		})
	}
}

func TestBudgetApprove_ShouldRecordDecision(t *testing.T) {
	budget := newBudget(t)

	err := budget.Approve()

	require.NoError(t, err)
	assert.True(t, budget.IsApproved())
	assert.NotNil(t, budget.DecidedAt())
}

func TestBudgetApprove_ShouldFailWhenAlreadyDecided(t *testing.T) {
	budget := newBudget(t)
	require.NoError(t, budget.Approve())

	err := budget.Approve()

	assert.ErrorIs(t, err, entity.ErrBudgetInvalidTransition)
}

func TestBudgetReject_ShouldStoreReason(t *testing.T) {
	budget := newBudget(t)

	err := budget.Reject("too expensive")

	require.NoError(t, err)
	assert.Equal(t, valueobject.BudgetStatusRejected, budget.Status())
	assert.Equal(t, "too expensive", budget.RejectionReason())
	assert.NotNil(t, budget.DecidedAt())
}

func TestBudgetReject_ShouldRequireReason(t *testing.T) {
	budget := newBudget(t)

	err := budget.Reject("")

	assert.ErrorIs(t, err, entity.ErrBudgetReasonRequired)
	assert.Equal(t, valueobject.BudgetStatusPendingApproval, budget.Status())
}

func TestBudgetReject_ShouldFailWhenAlreadyApproved(t *testing.T) {
	budget := newBudget(t)
	require.NoError(t, budget.Approve())

	err := budget.Reject("changed my mind")

	assert.ErrorIs(t, err, entity.ErrBudgetInvalidTransition)
	assert.Empty(t, budget.RejectionReason())
}

func TestBudgetCancel_ShouldCancelApprovedBudget(t *testing.T) {
	budget := newBudget(t)
	require.NoError(t, budget.Approve())

	err := budget.Cancel()

	require.NoError(t, err)
	assert.True(t, budget.IsCancelled())
}

func TestBudgetCancel_ShouldFailWhenAlreadyCancelled(t *testing.T) {
	budget := newBudget(t)
	require.NoError(t, budget.Cancel())

	err := budget.Cancel()

	assert.ErrorIs(t, err, entity.ErrBudgetInvalidTransition)
}

func TestRestoreBudget_ShouldKeepSnapshotValues(t *testing.T) {
	decidedAt := time.Now()
	snapshot := entity.BudgetSnapshot{
		ID:              "budget-1",
		OrderID:         "order-1",
		SagaID:          "saga-1",
		Customer:        customer,
		Status:          valueobject.BudgetStatusRejected,
		Items:           newItems(t),
		TotalAmount:     180,
		RejectionReason: "too expensive",
		DecidedAt:       &decidedAt,
		CreatedAt:       decidedAt,
		UpdatedAt:       decidedAt,
	}

	budget := entity.RestoreBudget(snapshot)

	assert.Equal(t, "budget-1", budget.ID())
	assert.Equal(t, valueobject.BudgetStatusRejected, budget.Status())
	assert.Equal(t, "rejected", budget.Status().String())
	assert.Equal(t, 180.0, budget.TotalAmount())
	assert.Equal(t, "too expensive", budget.RejectionReason())
	assert.Equal(t, &decidedAt, budget.DecidedAt())
}
