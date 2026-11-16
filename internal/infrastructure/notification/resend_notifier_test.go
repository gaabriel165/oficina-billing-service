package notification_test

import (
	"context"
	"testing"

	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/valueobject"
	"github.com/gabrielcamargo/oficina-billing-service/internal/infrastructure/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func approvedBudget(t *testing.T, customerName string) *entity.Budget {
	t.Helper()
	service, err := entity.NewBudgetItem(valueobject.BudgetItemKindService, "service-1", "Troca de óleo", 1, 120)
	require.NoError(t, err)
	part, err := entity.NewBudgetItem(valueobject.BudgetItemKindPart, "part-1", "Filtro de óleo", 2, 29.9)
	require.NoError(t, err)
	budget, err := entity.NewBudget("d1e6a7c4-2b1f-4f53-8a37-7d0a3c1b9e55", "saga-1",
		entity.BudgetCustomer{ID: "customer-1", Name: customerName, Email: "maria@example.com"},
		[]*entity.BudgetItem{service, part})
	require.NoError(t, err)
	require.NoError(t, budget.Approve())
	return budget
}

func requestedPayment(t *testing.T, budget *entity.Budget) *entity.Payment {
	t.Helper()
	payment, err := entity.NewPayment(budget, "fake")
	require.NoError(t, err)
	require.NoError(t, payment.AttachCheckout("pref-1", "https://checkout.example.com/pref-1"))
	return payment
}

func TestBuildBudgetEmail_ShouldListItemsAndTotalInReais(t *testing.T) {
	budget := approvedBudget(t, "Maria Silva")

	text := notification.BuildBudgetEmail(budget)

	assert.Contains(t, text, "Olá, Maria!")
	assert.Contains(t, text, "- Troca de óleo (x1): R$ 120,00")
	assert.Contains(t, text, "- Filtro de óleo (x2): R$ 59,80")
	assert.Contains(t, text, "Total: R$ 179,80")
	assert.Contains(t, text, "Orçamento: "+budget.ID())
}

func TestBuildBudgetEmail_ShouldUseGenericGreetingWhenNameIsMissing(t *testing.T) {
	text := notification.BuildBudgetEmail(approvedBudget(t, ""))

	assert.Contains(t, text, "Olá, cliente!")
}

func TestBuildPaymentEmail_ShouldIncludeAmountAndCheckoutLink(t *testing.T) {
	budget := approvedBudget(t, "Maria Silva")

	text := notification.BuildPaymentEmail(budget, requestedPayment(t, budget))

	assert.Contains(t, text, "Seu orçamento de R$ 179,80 foi aprovado")
	assert.Contains(t, text, "https://checkout.example.com/pref-1")
}

func TestResendNotifier_ShouldSkipSendingWithoutAPIKey(t *testing.T) {
	notifier := notification.NewResendNotifier("", "oficina@example.com")
	budget := approvedBudget(t, "Maria Silva")

	budgetErr := notifier.NotifyBudgetReady(context.Background(), budget)
	paymentErr := notifier.NotifyPaymentRequested(context.Background(), budget, requestedPayment(t, budget))

	assert.NoError(t, budgetErr)
	assert.NoError(t, paymentErr)
}
