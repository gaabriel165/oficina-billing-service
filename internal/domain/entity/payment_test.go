package entity_test

import (
	"testing"
	"time"

	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/valueobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newApprovedBudget(t *testing.T) *entity.Budget {
	t.Helper()
	budget := newBudget(t)
	require.NoError(t, budget.Approve())
	return budget
}

func newPayment(t *testing.T) *entity.Payment {
	t.Helper()
	payment, err := entity.NewPayment(newApprovedBudget(t), "fake")
	require.NoError(t, err)
	return payment
}

func TestNewPayment_ShouldStartPendingWithBudgetAmount(t *testing.T) {
	budget := newApprovedBudget(t)

	payment, err := entity.NewPayment(budget, "mercadopago")

	require.NoError(t, err)
	assert.NotEmpty(t, payment.ID())
	assert.Equal(t, budget.ID(), payment.BudgetID())
	assert.Equal(t, "order-1", payment.OrderID())
	assert.Equal(t, "saga-1", payment.SagaID())
	assert.Equal(t, 180.0, payment.Amount())
	assert.Equal(t, "mercadopago", payment.Provider())
	assert.True(t, payment.IsPending())
	assert.False(t, payment.IsConfirmed())
	assert.Empty(t, payment.CheckoutReference())
	assert.Empty(t, payment.ProviderPaymentID())
	assert.Empty(t, payment.FailureReason())
	assert.False(t, payment.CreatedAt().IsZero())
	assert.False(t, payment.UpdatedAt().IsZero())
}

func TestNewPayment_ShouldRequireBudget(t *testing.T) {
	_, err := entity.NewPayment(nil, "fake")

	assert.ErrorIs(t, err, entity.ErrPaymentBudgetRequired)
}

func TestNewPayment_ShouldRequireApprovedBudget(t *testing.T) {
	_, err := entity.NewPayment(newBudget(t), "fake")

	assert.ErrorIs(t, err, entity.ErrBudgetNotApproved)
}

func TestNewPayment_ShouldRejectBudgetWithoutAmount(t *testing.T) {
	budget := entity.RestoreBudget(entity.BudgetSnapshot{ID: "budget-1", Status: valueobject.BudgetStatusApproved})

	_, err := entity.NewPayment(budget, "fake")

	assert.ErrorIs(t, err, entity.ErrPaymentInvalidAmount)
}

func TestPaymentAttachCheckout_ShouldStoreReferenceAndURL(t *testing.T) {
	payment := newPayment(t)

	err := payment.AttachCheckout("pref-1", "https://checkout.example.com/pref-1")

	require.NoError(t, err)
	assert.Equal(t, "pref-1", payment.CheckoutReference())
	assert.Equal(t, "https://checkout.example.com/pref-1", payment.CheckoutURL())
}

func TestPaymentAttachCheckout_ShouldRequireReference(t *testing.T) {
	payment := newPayment(t)

	err := payment.AttachCheckout("", "https://checkout.example.com")

	assert.ErrorIs(t, err, entity.ErrPaymentCheckoutRequired)
}

func TestPaymentConfirm_ShouldStoreProviderPaymentID(t *testing.T) {
	payment := newPayment(t)

	err := payment.Confirm("mp-123")

	require.NoError(t, err)
	assert.True(t, payment.IsConfirmed())
	assert.Equal(t, "mp-123", payment.ProviderPaymentID())
}

func TestPaymentConfirm_ShouldFailWhenAlreadySettled(t *testing.T) {
	payment := newPayment(t)
	require.NoError(t, payment.Fail("mp-123", "rejected"))

	err := payment.Confirm("mp-456")

	assert.ErrorIs(t, err, entity.ErrPaymentInvalidTransition)
	assert.Equal(t, "mp-123", payment.ProviderPaymentID())
}

func TestPaymentFail_ShouldStoreReason(t *testing.T) {
	payment := newPayment(t)

	err := payment.Fail("mp-123", "cc_rejected_insufficient_amount")

	require.NoError(t, err)
	assert.Equal(t, valueobject.PaymentStatusFailed, payment.Status())
	assert.Equal(t, "failed", payment.Status().String())
	assert.Equal(t, "cc_rejected_insufficient_amount", payment.FailureReason())
}

func TestPaymentFail_ShouldFailWhenAlreadyConfirmed(t *testing.T) {
	payment := newPayment(t)
	require.NoError(t, payment.Confirm("mp-123"))

	err := payment.Fail("mp-123", "late rejection")

	assert.ErrorIs(t, err, entity.ErrPaymentInvalidTransition)
	assert.Empty(t, payment.FailureReason())
}

func TestPaymentRefund_ShouldRefundConfirmedPayment(t *testing.T) {
	payment := newPayment(t)
	require.NoError(t, payment.Confirm("mp-123"))

	err := payment.Refund()

	require.NoError(t, err)
	assert.Equal(t, valueobject.PaymentStatusRefunded, payment.Status())
}

func TestPaymentRefund_ShouldFailWhenNotConfirmed(t *testing.T) {
	payment := newPayment(t)

	err := payment.Refund()

	assert.ErrorIs(t, err, entity.ErrPaymentInvalidTransition)
}

func TestRestorePayment_ShouldKeepSnapshotValues(t *testing.T) {
	now := time.Now()
	snapshot := entity.PaymentSnapshot{
		ID:                "payment-1",
		BudgetID:          "budget-1",
		OrderID:           "order-1",
		SagaID:            "saga-1",
		Amount:            180,
		Status:            valueobject.PaymentStatusConfirmed,
		Provider:          "fake",
		CheckoutReference: "pref-1",
		CheckoutURL:       "https://checkout.example.com/pref-1",
		ProviderPaymentID: "mp-123",
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	payment := entity.RestorePayment(snapshot)

	assert.Equal(t, "payment-1", payment.ID())
	assert.Equal(t, "budget-1", payment.BudgetID())
	assert.True(t, payment.IsConfirmed())
	assert.Equal(t, "mp-123", payment.ProviderPaymentID())
	assert.Equal(t, now, payment.CreatedAt())
}
