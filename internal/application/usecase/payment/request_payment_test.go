package payment_test

import (
	"errors"
	"testing"

	"github.com/gabrielcamargo/oficina-billing-service/internal/application/mocks"
	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase/payment"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type requestPaymentMocks struct {
	budgetRepo    *mocks.MockBudgetRepository
	paymentRepo   *mocks.MockPaymentRepository
	gateway       *mocks.MockPaymentGatewayService
	notifications *mocks.MockNotificationService
}

func newRequestPaymentMocks(t *testing.T) requestPaymentMocks {
	return requestPaymentMocks{
		budgetRepo:    mocks.NewMockBudgetRepository(t),
		paymentRepo:   mocks.NewMockPaymentRepository(t),
		gateway:       mocks.NewMockPaymentGatewayService(t),
		notifications: mocks.NewMockNotificationService(t),
	}
}

func (m requestPaymentMocks) useCase() *payment.RequestPaymentUseCase {
	return payment.NewRequestPaymentUseCase(m.budgetRepo, m.paymentRepo, m.gateway, m.notifications)
}

func (m requestPaymentMocks) expectCheckout(t *testing.T, budget *entity.Budget) {
	m.paymentRepo.On("FindByOrderID", ctx, "order-1").Return(nil, repository.ErrPaymentNotFound)
	m.budgetRepo.On("FindByOrderID", ctx, "order-1").Return(budget, nil)
	m.gateway.On("Provider").Return("fake")
	m.gateway.On("CreateCheckout", ctx, mock.AnythingOfType("*entity.Payment"), budget).
		Return(repository.Checkout{Reference: "pref-1", URL: "https://checkout.example.com/pref-1"}, nil)
}

func TestRequestPayment_ShouldCreateCheckoutAndReplyToSaga(t *testing.T) {
	m := newRequestPaymentMocks(t)
	budget := approvedBudget(t)
	m.expectCheckout(t, budget)
	m.paymentRepo.On("Create", ctx, mock.AnythingOfType("*entity.Payment"), replyOfType(messaging.TypePaymentRequested)).Return(nil)
	m.notifications.On("NotifyPaymentRequested", ctx, budget, mock.AnythingOfType("*entity.Payment")).Return(nil)

	result, err := m.useCase().Execute(ctx, "order-1")

	require.NoError(t, err)
	assert.True(t, result.IsPending())
	assert.Equal(t, budget.TotalAmount(), result.Amount())
	assert.Equal(t, "pref-1", result.CheckoutReference())
	assert.Equal(t, "https://checkout.example.com/pref-1", result.CheckoutURL())
}

func TestRequestPayment_ShouldReturnExistingPaymentWhenCommandIsRepeated(t *testing.T) {
	m := newRequestPaymentMocks(t)
	existing := pendingPayment(t)
	m.paymentRepo.On("FindByOrderID", ctx, "order-1").Return(existing, nil)

	result, err := m.useCase().Execute(ctx, "order-1")

	require.NoError(t, err)
	assert.Equal(t, existing.ID(), result.ID())
}

func TestRequestPayment_ShouldSucceedWhenNotificationFails(t *testing.T) {
	m := newRequestPaymentMocks(t)
	budget := approvedBudget(t)
	m.expectCheckout(t, budget)
	m.paymentRepo.On("Create", ctx, mock.AnythingOfType("*entity.Payment"), replyOfType(messaging.TypePaymentRequested)).Return(nil)
	m.notifications.On("NotifyPaymentRequested", ctx, budget, mock.AnythingOfType("*entity.Payment")).Return(errors.New("resend unavailable"))

	result, err := m.useCase().Execute(ctx, "order-1")

	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRequestPayment_ShouldReturnErrorWhenPaymentLookupFails(t *testing.T) {
	m := newRequestPaymentMocks(t)
	m.paymentRepo.On("FindByOrderID", ctx, "order-1").Return(nil, errDatabase)

	_, err := m.useCase().Execute(ctx, "order-1")

	assert.ErrorIs(t, err, errDatabase)
}

func TestRequestPayment_ShouldReturnErrorWhenBudgetNotFound(t *testing.T) {
	m := newRequestPaymentMocks(t)
	m.paymentRepo.On("FindByOrderID", ctx, "order-1").Return(nil, repository.ErrPaymentNotFound)
	m.budgetRepo.On("FindByOrderID", ctx, "order-1").Return(nil, repository.ErrBudgetNotFound)

	_, err := m.useCase().Execute(ctx, "order-1")

	assert.ErrorIs(t, err, repository.ErrBudgetNotFound)
}

func TestRequestPayment_ShouldReturnErrorWhenBudgetIsNotApproved(t *testing.T) {
	m := newRequestPaymentMocks(t)
	m.paymentRepo.On("FindByOrderID", ctx, "order-1").Return(nil, repository.ErrPaymentNotFound)
	m.budgetRepo.On("FindByOrderID", ctx, "order-1").Return(pendingBudget(t), nil)
	m.gateway.On("Provider").Return("fake")

	_, err := m.useCase().Execute(ctx, "order-1")

	assert.ErrorIs(t, err, entity.ErrBudgetNotApproved)
}

func TestRequestPayment_ShouldReturnErrorWhenGatewayIsUnavailable(t *testing.T) {
	m := newRequestPaymentMocks(t)
	budget := approvedBudget(t)
	m.paymentRepo.On("FindByOrderID", ctx, "order-1").Return(nil, repository.ErrPaymentNotFound)
	m.budgetRepo.On("FindByOrderID", ctx, "order-1").Return(budget, nil)
	m.gateway.On("Provider").Return("fake")
	m.gateway.On("CreateCheckout", ctx, mock.AnythingOfType("*entity.Payment"), budget).
		Return(repository.Checkout{}, repository.ErrPaymentGatewayUnavailable)

	_, err := m.useCase().Execute(ctx, "order-1")

	assert.ErrorIs(t, err, repository.ErrPaymentGatewayUnavailable)
}

func TestRequestPayment_ShouldReturnErrorWhenGatewayReturnsNoReference(t *testing.T) {
	m := newRequestPaymentMocks(t)
	budget := approvedBudget(t)
	m.paymentRepo.On("FindByOrderID", ctx, "order-1").Return(nil, repository.ErrPaymentNotFound)
	m.budgetRepo.On("FindByOrderID", ctx, "order-1").Return(budget, nil)
	m.gateway.On("Provider").Return("fake")
	m.gateway.On("CreateCheckout", ctx, mock.AnythingOfType("*entity.Payment"), budget).Return(repository.Checkout{}, nil)

	_, err := m.useCase().Execute(ctx, "order-1")

	assert.ErrorIs(t, err, entity.ErrPaymentCheckoutRequired)
}

func TestRequestPayment_ShouldReturnErrorWhenSaveFails(t *testing.T) {
	m := newRequestPaymentMocks(t)
	m.expectCheckout(t, approvedBudget(t))
	m.paymentRepo.On("Create", ctx, mock.AnythingOfType("*entity.Payment"), replyOfType(messaging.TypePaymentRequested)).Return(errDatabase)

	_, err := m.useCase().Execute(ctx, "order-1")

	assert.ErrorIs(t, err, errDatabase)
}
