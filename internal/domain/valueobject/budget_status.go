package valueobject

type BudgetStatus string

const (
	BudgetStatusPendingApproval BudgetStatus = "pending_approval"
	BudgetStatusApproved        BudgetStatus = "approved"
	BudgetStatusRejected        BudgetStatus = "rejected"
	BudgetStatusCancelled       BudgetStatus = "cancelled"
)

var allowedBudgetTransitions = map[BudgetStatus][]BudgetStatus{
	BudgetStatusPendingApproval: {BudgetStatusApproved, BudgetStatusRejected, BudgetStatusCancelled},
	BudgetStatusApproved:        {BudgetStatusCancelled},
	BudgetStatusRejected:        {BudgetStatusCancelled},
	BudgetStatusCancelled:       {},
}

func (s BudgetStatus) CanTransitionTo(next BudgetStatus) bool {
	for _, status := range allowedBudgetTransitions[s] {
		if status == next {
			return true
		}
	}
	return false
}

func (s BudgetStatus) String() string {
	return string(s)
}
