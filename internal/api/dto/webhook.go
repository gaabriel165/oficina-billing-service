package dto

type MercadoPagoNotification struct {
	Type   string                      `json:"type"`
	Action string                      `json:"action"`
	Data   MercadoPagoNotificationData `json:"data"`
}

type MercadoPagoNotificationData struct {
	ID string `json:"id"`
}

type WebhookResponse struct {
	Status string `json:"status"`
}
