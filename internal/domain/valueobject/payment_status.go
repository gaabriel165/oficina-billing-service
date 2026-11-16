package valueobject

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusConfirmed PaymentStatus = "confirmed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

var allowedPaymentTransitions = map[PaymentStatus][]PaymentStatus{
	PaymentStatusPending:   {PaymentStatusConfirmed, PaymentStatusFailed},
	PaymentStatusConfirmed: {PaymentStatusRefunded},
	PaymentStatusFailed:    {},
	PaymentStatusRefunded:  {},
}

func (s PaymentStatus) CanTransitionTo(next PaymentStatus) bool {
	for _, status := range allowedPaymentTransitions[s] {
		if status == next {
			return true
		}
	}
	return false
}

func (s PaymentStatus) String() string {
	return string(s)
}
