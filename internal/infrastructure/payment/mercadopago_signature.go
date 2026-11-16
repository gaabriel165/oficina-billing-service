package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

var ErrInvalidWebhookSignature = errors.New("invalid webhook signature")

func ValidateMercadoPagoSignature(signatureHeader, requestID, dataID, secret string) error {
	timestamp, received := parseSignatureHeader(signatureHeader)
	if secret == "" || timestamp == "" || received == "" {
		return ErrInvalidWebhookSignature
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signatureManifest(dataID, requestID, timestamp)))
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(received)) {
		return ErrInvalidWebhookSignature
	}
	return nil
}

func parseSignatureHeader(header string) (timestamp, signature string) {
	for _, part := range strings.Split(header, ",") {
		key, value, found := strings.Cut(part, "=")
		if !found {
			continue
		}
		switch strings.TrimSpace(key) {
		case "ts":
			timestamp = strings.TrimSpace(value)
		case "v1":
			signature = strings.TrimSpace(value)
		}
	}
	return timestamp, signature
}

func signatureManifest(dataID, requestID, timestamp string) string {
	var manifest strings.Builder
	if dataID != "" {
		manifest.WriteString("id:" + strings.ToLower(dataID) + ";")
	}
	if requestID != "" {
		manifest.WriteString("request-id:" + requestID + ";")
	}
	manifest.WriteString("ts:" + timestamp + ";")
	return manifest.String()
}
