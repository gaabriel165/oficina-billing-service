package repository

import "errors"

var (
	ErrBudgetNotFound  = errors.New("budget not found")
	ErrPaymentNotFound = errors.New("payment not found")
)
