package usecase

import "github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"

func Reply(messageType, sagaID, orderID string, payload any) (messaging.OutgoingMessage, error) {
	message, err := messaging.NewMessage(messageType, sagaID, orderID, payload)
	if err != nil {
		return messaging.OutgoingMessage{}, err
	}
	return messaging.OutgoingMessage{Destination: messaging.DestinationSagaReplies, Message: message}, nil
}
