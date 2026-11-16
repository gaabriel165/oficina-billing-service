package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gabrielcamargo/oficina-billing-service/internal/api/dto"
	"github.com/gabrielcamargo/oficina-billing-service/internal/api/response"
	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase/payment"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/repository"
	"github.com/gabrielcamargo/oficina-billing-service/internal/infrastructure/observability"
	"github.com/gin-gonic/gin"
)

const (
	notificationTypePayment = "payment"

	eventWebhookIgnored = "payment.webhook_ignored"

	webhookStatusProcessed = "processed"
	webhookStatusIgnored   = "ignored"
)

type MercadoPagoWebhookHandler struct {
	notificationUC *payment.HandleGatewayNotificationUseCase
}

func NewMercadoPagoWebhookHandler(notificationUC *payment.HandleGatewayNotificationUseCase) *MercadoPagoWebhookHandler {
	return &MercadoPagoWebhookHandler{notificationUC: notificationUC}
}

func (h *MercadoPagoWebhookHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/mercadopago", h.Receive)
}

// Receive godoc
// @Summary      Receive a Mercado Pago payment notification
// @Description  Validates the x-signature header, reads the payment from Mercado Pago and settles the local payment
// @Tags         webhooks
// @Accept       json
// @Produce      json
// @Param        data.id      query     string                       true  "Mercado Pago payment ID"
// @Param        x-signature  header    string                       true  "ts and v1 signature"
// @Param        body         body      dto.MercadoPagoNotification  true  "Notification"
// @Success      200          {object}  dto.WebhookResponse
// @Failure      401          {object}  response.ErrorResponse
// @Failure      502          {object}  response.ErrorResponse
// @Router       /webhooks/mercadopago [post]
func (h *MercadoPagoWebhookHandler) Receive(c *gin.Context) {
	var notification dto.MercadoPagoNotification
	if err := c.ShouldBindJSON(&notification); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	providerPaymentID := c.Query("data.id")
	if notification.Type != notificationTypePayment || providerPaymentID == "" {
		h.ignore(c, notification, "not a payment notification")
		return
	}

	settled, err := h.notificationUC.Execute(c.Request.Context(), providerPaymentID)
	if errors.Is(err, repository.ErrPaymentNotFound) {
		h.ignore(c, notification, "payment does not belong to this service")
		return
	}
	if err != nil {
		observability.Failure(c.Request.Context(), eventPaymentFailed, err, slog.String("provider_payment_id", providerPaymentID))
		response.Error(c, err)
		return
	}
	if settled == nil {
		h.ignore(c, notification, "payment is still pending")
		return
	}

	observability.Event(c.Request.Context(), eventPaymentSettled,
		slog.String("payment_id", settled.ID()),
		slog.String("order_id", settled.OrderID()),
		slog.String("saga_id", settled.SagaID()),
		slog.String("status", settled.Status().String()),
		slog.Float64("amount", settled.Amount()),
		slog.String("provider_payment_id", providerPaymentID),
	)
	c.JSON(http.StatusOK, dto.WebhookResponse{Status: webhookStatusProcessed})
}

func (h *MercadoPagoWebhookHandler) ignore(c *gin.Context, notification dto.MercadoPagoNotification, reason string) {
	observability.Event(c.Request.Context(), eventWebhookIgnored,
		slog.String("type", notification.Type),
		slog.String("action", notification.Action),
		slog.String("reason", reason),
	)
	c.JSON(http.StatusOK, dto.WebhookResponse{Status: webhookStatusIgnored})
}
