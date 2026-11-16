package handler

import (
	"log/slog"
	"net/http"

	"github.com/gabrielcamargo/oficina-billing-service/internal/api/dto"
	"github.com/gabrielcamargo/oficina-billing-service/internal/api/response"
	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase/payment"
	"github.com/gabrielcamargo/oficina-billing-service/internal/infrastructure/observability"
	"github.com/gin-gonic/gin"
)

const (
	eventPaymentSettled = "payment.settled"
	eventPaymentFailed  = "payment.failed"
)

type PaymentHandler struct {
	getUC    *payment.GetPaymentUseCase
	settleUC *payment.SettlePaymentUseCase
}

func NewPaymentHandler(getUC *payment.GetPaymentUseCase, settleUC *payment.SettlePaymentUseCase) *PaymentHandler {
	return &PaymentHandler{getUC: getUC, settleUC: settleUC}
}

func (h *PaymentHandler) RegisterOperatorRoutes(router *gin.RouterGroup) {
	router.GET("/payments/:id", h.Get)
}

func (h *PaymentHandler) RegisterSimulationRoutes(router *gin.RouterGroup) {
	router.POST("/payments/:id/simulate", h.Simulate)
}

// Get godoc
// @Summary      Get a payment by ID
// @Tags         payments
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Payment ID"
// @Success      200  {object}  dto.PaymentResponse
// @Failure      404  {object}  response.ErrorResponse
// @Router       /payments/{id} [get]
func (h *PaymentHandler) Get(c *gin.Context) {
	result, err := h.getUC.Execute(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ToPaymentResponse(result))
}

// Simulate godoc
// @Summary      Simulate the payment outcome; only available with the fake payment gateway
// @Tags         payments
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path      string                      true  "Payment ID"
// @Param        body  body      dto.SimulatePaymentRequest  true  "Outcome"
// @Success      200   {object}  dto.PaymentResponse
// @Failure      400   {object}  response.ErrorResponse
// @Failure      404   {object}  response.ErrorResponse
// @Router       /payments/{id}/simulate [post]
func (h *PaymentHandler) Simulate(c *gin.Context) {
	var req dto.SimulatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.settleUC.Execute(c.Request.Context(), payment.SettlePaymentInput{
		PaymentID:         c.Param("id"),
		ProviderPaymentID: "simulated-" + c.Param("id"),
		Approved:          req.Approved,
		Reason:            req.Reason,
	})
	if err != nil {
		observability.Failure(c.Request.Context(), eventPaymentFailed, err, slog.String("payment_id", c.Param("id")))
		response.Error(c, err)
		return
	}

	observability.Event(c.Request.Context(), eventPaymentSettled,
		slog.String("payment_id", result.ID()),
		slog.String("order_id", result.OrderID()),
		slog.String("saga_id", result.SagaID()),
		slog.String("status", result.Status().String()),
		slog.Float64("amount", result.Amount()),
	)
	c.JSON(http.StatusOK, dto.ToPaymentResponse(result))
}
