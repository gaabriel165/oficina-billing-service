package response

import (
	"errors"
	"net/http"

	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func Error(c *gin.Context, err error) {
	status := statusFromError(err)
	if status == http.StatusInternalServerError {
		c.JSON(status, ErrorResponse{Error: "internal server error"})
		return
	}
	c.JSON(status, ErrorResponse{Error: err.Error()})
}

func statusFromError(err error) int {
	notFound := []error{
		repository.ErrBudgetNotFound,
		repository.ErrPaymentNotFound,
	}
	for _, e := range notFound {
		if errors.Is(err, e) {
			return http.StatusNotFound
		}
	}

	unprocessable := []error{
		entity.ErrBudgetInvalidTransition,
		entity.ErrBudgetReasonRequired,
		entity.ErrBudgetNotApproved,
		entity.ErrPaymentInvalidTransition,
		entity.ErrPaymentInvalidAmount,
	}
	for _, e := range unprocessable {
		if errors.Is(err, e) {
			return http.StatusUnprocessableEntity
		}
	}

	if errors.Is(err, usecase.ErrForbidden) {
		return http.StatusForbidden
	}

	if errors.Is(err, repository.ErrPaymentGatewayUnavailable) {
		return http.StatusBadGateway
	}

	return http.StatusInternalServerError
}
