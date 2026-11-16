package middleware

import (
	"net/http"

	"github.com/gabrielcamargo/oficina-billing-service/internal/infrastructure/payment"
	"github.com/gin-gonic/gin"
)

const (
	headerMercadoPagoSignature = "x-signature"
	headerMercadoPagoRequestID = "x-request-id"
	queryMercadoPagoDataID     = "data.id"
)

func MercadoPagoSignature(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := payment.ValidateMercadoPagoSignature(
			c.GetHeader(headerMercadoPagoSignature),
			c.GetHeader(headerMercadoPagoRequestID),
			c.Query(queryMercadoPagoDataID),
			secret,
		)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid webhook signature"})
			return
		}
		c.Next()
	}
}
