package payment_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/gabrielcamargo/oficina-billing-service/internal/infrastructure/payment"
	"github.com/stretchr/testify/assert"
)

const webhookSecret = "webhook-secret"

func sign(manifest string) string {
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write([]byte(manifest))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestValidateMercadoPagoSignature_ShouldAcceptSignatureOfTheManifest(t *testing.T) {
	header := "ts=1704908010,v1=" + sign("id:999999999;request-id:req-1;ts:1704908010;")

	err := payment.ValidateMercadoPagoSignature(header, "req-1", "999999999", webhookSecret)

	assert.NoError(t, err)
}

func TestValidateMercadoPagoSignature_ShouldLowercaseAlphanumericIDs(t *testing.T) {
	header := "ts=1704908010, v1=" + sign("id:abc123;request-id:req-1;ts:1704908010;")

	err := payment.ValidateMercadoPagoSignature(header, "req-1", "ABC123", webhookSecret)

	assert.NoError(t, err)
}

func TestValidateMercadoPagoSignature_ShouldOmitMissingPartsFromManifest(t *testing.T) {
	header := "ts=1704908010,v1=" + sign("ts:1704908010;")

	err := payment.ValidateMercadoPagoSignature(header, "", "", webhookSecret)

	assert.NoError(t, err)
}

func TestValidateMercadoPagoSignature_ShouldRejectInvalidSignatures(t *testing.T) {
	valid := sign("id:999999999;request-id:req-1;ts:1704908010;")
	cases := map[string]struct {
		header    string
		requestID string
		dataID    string
		secret    string
	}{
		"tampered data id":  {"ts=1704908010,v1=" + valid, "req-1", "111111111", webhookSecret},
		"another request":   {"ts=1704908010,v1=" + valid, "req-2", "999999999", webhookSecret},
		"another timestamp": {"ts=1704908011,v1=" + valid, "req-1", "999999999", webhookSecret},
		"another secret":    {"ts=1704908010,v1=" + valid, "req-1", "999999999", "other-secret"},
		"missing secret":    {"ts=1704908010,v1=" + valid, "req-1", "999999999", ""},
		"missing header":    {"", "req-1", "999999999", webhookSecret},
		"missing hash":      {"ts=1704908010", "req-1", "999999999", webhookSecret},
		"malformed header":  {"garbage", "req-1", "999999999", webhookSecret},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := payment.ValidateMercadoPagoSignature(tc.header, tc.requestID, tc.dataID, tc.secret)

			assert.ErrorIs(t, err, payment.ErrInvalidWebhookSignature)
		})
	}
}
