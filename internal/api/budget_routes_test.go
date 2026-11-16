package api_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gabrielcamargo/oficina-billing-service/configs"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestListBudgets_ShouldRequireAuthentication(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)

	recorder := environment.request(t, http.MethodGet, "/api/v1/budgets", "", "", "")

	assertStatus(t, http.StatusUnauthorized, recorder)
}

func TestListBudgets_ShouldReturnAllBudgetsToOperator(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)
	environment.budgetRepo.On("FindAll", mock.Anything).Return([]*entity.Budget{pendingBudget(t)}, nil)

	recorder := environment.request(t, http.MethodGet, "/api/v1/budgets", roleOperator, "operator-1", "")

	assertStatus(t, http.StatusOK, recorder)
	assert.Contains(t, recorder.Body.String(), `"status":"pending_approval"`)
	assert.Contains(t, recorder.Body.String(), `"total_amount":120`)
}

func TestListBudgets_ShouldReturnOnlyOwnBudgetsToCustomer(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)
	environment.budgetRepo.On("FindByCustomerID", mock.Anything, "customer-1").Return([]*entity.Budget{}, nil)

	recorder := environment.request(t, http.MethodGet, "/api/v1/budgets", roleCustomer, "customer-1", "")

	assertStatus(t, http.StatusOK, recorder)
	assert.JSONEq(t, `[]`, recorder.Body.String())
}

func TestListBudgets_ShouldReturnNotFoundWhenOrderHasNoBudget(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)
	environment.budgetRepo.On("FindByOrderID", mock.Anything, "order-9").Return(nil, repository.ErrBudgetNotFound)

	recorder := environment.request(t, http.MethodGet, "/api/v1/budgets?order_id=order-9", roleOperator, "operator-1", "")

	assertStatus(t, http.StatusNotFound, recorder)
}

func TestGetBudget_ShouldReturnBudgetWithItems(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)
	budget := pendingBudget(t)
	environment.budgetRepo.On("FindByID", mock.Anything, budget.ID()).Return(budget, nil)

	recorder := environment.request(t, http.MethodGet, "/api/v1/budgets/"+budget.ID(), roleCustomer, "customer-1", "")

	assertStatus(t, http.StatusOK, recorder)
	assert.Contains(t, recorder.Body.String(), `"description":"Troca de óleo"`)
	assert.Contains(t, recorder.Body.String(), `"kind":"service"`)
}

func TestGetBudget_ShouldForbidAnotherCustomer(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)
	budget := pendingBudget(t)
	environment.budgetRepo.On("FindByID", mock.Anything, budget.ID()).Return(budget, nil)

	recorder := environment.request(t, http.MethodGet, "/api/v1/budgets/"+budget.ID(), roleCustomer, "customer-2", "")

	assertStatus(t, http.StatusForbidden, recorder)
}

func TestGetBudget_ShouldHideInternalErrorDetails(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)
	environment.budgetRepo.On("FindByID", mock.Anything, "budget-1").Return(nil, errors.New("connection refused"))

	recorder := environment.request(t, http.MethodGet, "/api/v1/budgets/budget-1", roleOperator, "operator-1", "")

	assertStatus(t, http.StatusInternalServerError, recorder)
	assert.JSONEq(t, `{"error":"internal server error"}`, recorder.Body.String())
}

func TestGetBudgetPayment_ShouldReturnCheckoutLink(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)
	budget := pendingBudget(t)
	environment.budgetRepo.On("FindByID", mock.Anything, budget.ID()).Return(budget, nil)
	environment.paymentRepo.On("FindByOrderID", mock.Anything, "order-1").Return(pendingPayment(t), nil)

	recorder := environment.request(t, http.MethodGet, "/api/v1/budgets/"+budget.ID()+"/payment", roleCustomer, "customer-1", "")

	assertStatus(t, http.StatusOK, recorder)
	assert.Contains(t, recorder.Body.String(), `"checkout_url":"https://checkout.example.com/pref-1"`)
}

func TestGetBudgetPayment_ShouldReturnNotFoundWhenPaymentWasNotRequested(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)
	budget := pendingBudget(t)
	environment.budgetRepo.On("FindByID", mock.Anything, budget.ID()).Return(budget, nil)
	environment.paymentRepo.On("FindByOrderID", mock.Anything, "order-1").Return(nil, repository.ErrPaymentNotFound)

	recorder := environment.request(t, http.MethodGet, "/api/v1/budgets/"+budget.ID()+"/payment", roleCustomer, "customer-1", "")

	assertStatus(t, http.StatusNotFound, recorder)
}

func TestGetBudgetPayment_ShouldForbidAnotherCustomer(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)
	budget := pendingBudget(t)
	environment.budgetRepo.On("FindByID", mock.Anything, budget.ID()).Return(budget, nil)

	recorder := environment.request(t, http.MethodGet, "/api/v1/budgets/"+budget.ID()+"/payment", roleCustomer, "customer-2", "")

	assertStatus(t, http.StatusForbidden, recorder)
}

func TestApproveBudget_ShouldReturnApprovedBudget(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)
	budget := pendingBudget(t)
	environment.budgetRepo.On("FindByID", mock.Anything, budget.ID()).Return(budget, nil)
	environment.budgetRepo.On("Update", mock.Anything, budget, replyOfType(messaging.TypeBudgetApproved)).Return(nil)

	recorder := environment.request(t, http.MethodPatch, "/api/v1/budgets/"+budget.ID()+"/approve", roleCustomer, "customer-1", "")

	assertStatus(t, http.StatusOK, recorder)
	assert.Contains(t, recorder.Body.String(), `"status":"approved"`)
}

func TestApproveBudget_ShouldReturnUnprocessableWhenAlreadyDecided(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)
	budget := pendingBudget(t)
	require.NoError(t, budget.Approve())
	environment.budgetRepo.On("FindByID", mock.Anything, budget.ID()).Return(budget, nil)

	recorder := environment.request(t, http.MethodPatch, "/api/v1/budgets/"+budget.ID()+"/approve", roleCustomer, "customer-1", "")

	assertStatus(t, http.StatusUnprocessableEntity, recorder)
}

func TestRejectBudget_ShouldReturnRejectedBudget(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)
	budget := pendingBudget(t)
	environment.budgetRepo.On("FindByID", mock.Anything, budget.ID()).Return(budget, nil)
	environment.budgetRepo.On("Update", mock.Anything, budget, replyOfType(messaging.TypeBudgetRejected)).Return(nil)

	recorder := environment.request(t, http.MethodPatch, "/api/v1/budgets/"+budget.ID()+"/reject", roleCustomer, "customer-1", `{"reason":"too expensive"}`)

	assertStatus(t, http.StatusOK, recorder)
	assert.Contains(t, recorder.Body.String(), `"rejection_reason":"too expensive"`)
}

func TestRejectBudget_ShouldReturnBadRequestWithoutReason(t *testing.T) {
	environment := newTestEnvironment(t, configs.PaymentGatewayFake)

	recorder := environment.request(t, http.MethodPatch, "/api/v1/budgets/budget-1/reject", roleCustomer, "customer-1", `{}`)

	assertStatus(t, http.StatusBadRequest, recorder)
}
