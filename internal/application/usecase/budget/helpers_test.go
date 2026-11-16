package budget_test

import (
	"context"
	"errors"
	"testing"

	"github.com/gabrielcamargo/oficina-billing-service/internal/application/usecase/budget"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/messaging"
	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/valueobject"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	ctx           = context.Background()
	errDatabase   = errors.New("database unavailable")
	owner         = budget.Requester{ID: "customer-1", IsCustomer: true}
	otherCustomer = budget.Requester{ID: "customer-2", IsCustomer: true}
	operator      = budget.Requester{ID: "operator-1"}
)

func pendingBudget(t *testing.T) *entity.Budget {
	t.Helper()
	item, err := entity.NewBudgetItem(valueobject.BudgetItemKindService, "service-1", "Troca de óleo", 1, 120)
	require.NoError(t, err)
	created, err := entity.NewBudget("order-1", "saga-1", entity.BudgetCustomer{ID: "customer-1", Name: "Maria Silva", Email: "maria@example.com"}, []*entity.BudgetItem{item})
	require.NoError(t, err)
	return created
}

func replyOfType(messageType string) any {
	return mock.MatchedBy(func(outgoing []messaging.OutgoingMessage) bool {
		return len(outgoing) == 1 &&
			outgoing[0].Destination == messaging.DestinationSagaReplies &&
			outgoing[0].Message.Type == messageType &&
			outgoing[0].Message.SagaID == "saga-1" &&
			outgoing[0].Message.OrderID == "order-1"
	})
}
