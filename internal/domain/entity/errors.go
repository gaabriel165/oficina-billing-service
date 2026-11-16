package entity

import "errors"

var (
	ErrBudgetOrderRequired      = errors.New("budget service order id is required")
	ErrBudgetSagaRequired       = errors.New("budget saga id is required")
	ErrBudgetCustomerRequired   = errors.New("budget customer id is required")
	ErrBudgetNoItems            = errors.New("budget must have at least one service or part")
	ErrBudgetInvalidTransition  = errors.New("budget status transition not allowed")
	ErrBudgetItemDescription    = errors.New("budget item description is required")
	ErrBudgetItemInvalidQty     = errors.New("budget item quantity must be greater than zero")
	ErrBudgetItemInvalidPrice   = errors.New("budget item unit price must be greater than zero")
	ErrBudgetReasonRequired     = errors.New("budget rejection reason is required")
	ErrBudgetNotApproved        = errors.New("budget must be approved before payment")
	ErrPaymentBudgetRequired    = errors.New("payment budget id is required")
	ErrPaymentInvalidAmount     = errors.New("payment amount must be greater than zero")
	ErrPaymentInvalidTransition = errors.New("payment status transition not allowed")
	ErrPaymentCheckoutRequired  = errors.New("payment checkout reference is required")
)
