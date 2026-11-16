package handler

import (
	"log/slog"
	"net/http"

	"github.com/gabrielcamargo/oficina-billing-service/internal/api/dto"
	"github.com/gabrielcamargo/oficina-billing-service/internal/api/middleware"
	"github.com/gabrielcamargo/oficina-billing-service/internal/api/response"
	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase/budget"
	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase/payment"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/infrastructure/observability"
	"github.com/gin-gonic/gin"
)

const (
	eventBudgetDecided = "budget.decided"
	eventBudgetFailed  = "budget.failed"
)

type BudgetHandler struct {
	approveUC *budget.ApproveBudgetUseCase
	rejectUC  *budget.RejectBudgetUseCase
	getUC     *budget.GetBudgetUseCase
	listUC    *budget.ListBudgetsUseCase
	paymentUC *payment.GetPaymentUseCase
}

func NewBudgetHandler(
	approveUC *budget.ApproveBudgetUseCase,
	rejectUC *budget.RejectBudgetUseCase,
	getUC *budget.GetBudgetUseCase,
	listUC *budget.ListBudgetsUseCase,
	paymentUC *payment.GetPaymentUseCase,
) *BudgetHandler {
	return &BudgetHandler{approveUC: approveUC, rejectUC: rejectUC, getUC: getUC, listUC: listUC, paymentUC: paymentUC}
}

func (h *BudgetHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/budgets", h.List)
	router.GET("/budgets/:id", h.Get)
	router.GET("/budgets/:id/payment", h.GetPayment)
	router.PATCH("/budgets/:id/approve", h.Approve)
	router.PATCH("/budgets/:id/reject", h.Reject)
}

func requesterFrom(c *gin.Context) budget.Requester {
	subject, role := middleware.Principal(c)
	return budget.Requester{ID: subject, IsCustomer: role == middleware.RoleCustomer}
}

// List godoc
// @Summary      List budgets; customers only see their own
// @Tags         budgets
// @Security     BearerAuth
// @Produce      json
// @Param        order_id  query     string  false  "Service order ID"
// @Success      200       {array}   dto.BudgetResponse
// @Failure      404       {object}  response.ErrorResponse
// @Router       /budgets [get]
func (h *BudgetHandler) List(c *gin.Context) {
	budgets, err := h.listUC.Execute(c.Request.Context(), c.Query("order_id"), requesterFrom(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	responses := make([]dto.BudgetResponse, 0, len(budgets))
	for _, item := range budgets {
		responses = append(responses, dto.ToBudgetResponse(item))
	}
	c.JSON(http.StatusOK, responses)
}

// Get godoc
// @Summary      Get a budget by ID
// @Tags         budgets
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Budget ID"
// @Success      200  {object}  dto.BudgetResponse
// @Failure      403  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Router       /budgets/{id} [get]
func (h *BudgetHandler) Get(c *gin.Context) {
	result, err := h.getUC.Execute(c.Request.Context(), c.Param("id"), requesterFrom(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ToBudgetResponse(result))
}

// GetPayment godoc
// @Summary      Get the payment of a budget, with the checkout link
// @Tags         budgets
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Budget ID"
// @Success      200  {object}  dto.PaymentResponse
// @Failure      403  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Router       /budgets/{id}/payment [get]
func (h *BudgetHandler) GetPayment(c *gin.Context) {
	result, err := h.getUC.Execute(c.Request.Context(), c.Param("id"), requesterFrom(c))
	if err != nil {
		response.Error(c, err)
		return
	}

	budgetPayment, err := h.paymentUC.ByOrder(c.Request.Context(), result.OrderID())
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ToPaymentResponse(budgetPayment))
}

// Approve godoc
// @Summary      Approve a budget; the payment is requested next
// @Tags         budgets
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Budget ID"
// @Success      200  {object}  dto.BudgetResponse
// @Failure      403  {object}  response.ErrorResponse
// @Failure      404  {object}  response.ErrorResponse
// @Failure      422  {object}  response.ErrorResponse
// @Router       /budgets/{id}/approve [patch]
func (h *BudgetHandler) Approve(c *gin.Context) {
	result, err := h.approveUC.Execute(c.Request.Context(), c.Param("id"), requesterFrom(c))
	respondBudgetDecision(c, "approve", result, err)
}

// Reject godoc
// @Summary      Reject a budget, cancelling the service order
// @Tags         budgets
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path      string                   true  "Budget ID"
// @Param        body  body      dto.RejectBudgetRequest  true  "Rejection reason"
// @Success      200   {object}  dto.BudgetResponse
// @Failure      400   {object}  response.ErrorResponse
// @Failure      403   {object}  response.ErrorResponse
// @Failure      404   {object}  response.ErrorResponse
// @Failure      422   {object}  response.ErrorResponse
// @Router       /budgets/{id}/reject [patch]
func (h *BudgetHandler) Reject(c *gin.Context) {
	var req dto.RejectBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.rejectUC.Execute(c.Request.Context(), budget.RejectBudgetInput{
		BudgetID:  c.Param("id"),
		Reason:    req.Reason,
		Requester: requesterFrom(c),
	})
	respondBudgetDecision(c, "reject", result, err)
}

func respondBudgetDecision(c *gin.Context, action string, result *entity.Budget, err error) {
	if err != nil {
		observability.Failure(c.Request.Context(), eventBudgetFailed, err,
			slog.String("action", action),
			slog.String("budget_id", c.Param("id")),
		)
		response.Error(c, err)
		return
	}

	observability.Event(c.Request.Context(), eventBudgetDecided,
		slog.String("action", action),
		slog.String("budget_id", result.ID()),
		slog.String("order_id", result.OrderID()),
		slog.String("saga_id", result.SagaID()),
		slog.String("status", result.Status().String()),
		slog.Float64("total_amount", result.TotalAmount()),
	)
	c.JSON(http.StatusOK, dto.ToBudgetResponse(result))
}
