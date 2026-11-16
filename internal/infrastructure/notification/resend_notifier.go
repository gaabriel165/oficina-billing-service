package notification

import (
	"context"
	"fmt"
	"strings"

	"github.com/gabrielcamargo/oficina-billing-service/internal/domain/entity"
	"github.com/resend/resend-go/v3"
)

type ResendNotifier struct {
	client  *resend.Client
	from    string
	enabled bool
}

func NewResendNotifier(apiKey, from string) *ResendNotifier {
	return &ResendNotifier{client: resend.NewClient(apiKey), from: from, enabled: apiKey != ""}
}

func (n *ResendNotifier) NotifyBudgetReady(ctx context.Context, budget *entity.Budget) error {
	subject := fmt.Sprintf("Oficina Mecânica · Orçamento da OS %s", shortID(budget.OrderID()))
	return n.send(ctx, budget.Customer().Email, subject, BuildBudgetEmail(budget))
}

func (n *ResendNotifier) NotifyPaymentRequested(ctx context.Context, budget *entity.Budget, payment *entity.Payment) error {
	subject := fmt.Sprintf("Oficina Mecânica · Pagamento da OS %s", shortID(budget.OrderID()))
	return n.send(ctx, budget.Customer().Email, subject, BuildPaymentEmail(budget, payment))
}

func (n *ResendNotifier) send(ctx context.Context, recipient, subject, text string) error {
	if !n.enabled || recipient == "" {
		return nil
	}

	_, err := n.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    n.from,
		To:      []string{recipient},
		Subject: subject,
		Text:    text,
	})
	return err
}

func BuildBudgetEmail(budget *entity.Budget) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "Olá, %s!\n\n", firstName(budget.Customer().Name))
	builder.WriteString("O orçamento da sua ordem de serviço está pronto:\n\n")
	for _, item := range budget.Items() {
		fmt.Fprintf(&builder, "- %s (x%d): %s\n", item.Description(), item.Quantity(), formatBRL(item.TotalPrice()))
	}
	fmt.Fprintf(&builder, "\nTotal: %s\n\n", formatBRL(budget.TotalAmount()))
	fmt.Fprintf(&builder, "Orçamento: %s\n", budget.ID())
	builder.WriteString("Assim que você aprovar, enviaremos o link de pagamento.\n\n")
	builder.WriteString("Oficina Mecânica — este é um e-mail automático.")
	return builder.String()
}

func BuildPaymentEmail(budget *entity.Budget, payment *entity.Payment) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "Olá, %s!\n\n", firstName(budget.Customer().Name))
	fmt.Fprintf(&builder, "Seu orçamento de %s foi aprovado. Para iniciarmos o serviço, conclua o pagamento:\n\n", formatBRL(payment.Amount()))
	fmt.Fprintf(&builder, "%s\n\n", payment.CheckoutURL())
	builder.WriteString("Oficina Mecânica — este é um e-mail automático.")
	return builder.String()
}

func firstName(name string) string {
	fields := strings.Fields(name)
	if len(fields) == 0 {
		return "cliente"
	}
	return fields[0]
}

func shortID(id string) string {
	if len(id) < 8 {
		return id
	}
	return id[:8]
}

func formatBRL(value float64) string {
	return "R$ " + strings.Replace(fmt.Sprintf("%.2f", value), ".", ",", 1)
}
