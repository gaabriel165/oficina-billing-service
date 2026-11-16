package valueobject

type BudgetItemKind string

const (
	BudgetItemKindService BudgetItemKind = "service"
	BudgetItemKindPart    BudgetItemKind = "part"
)

func (k BudgetItemKind) String() string {
	return string(k)
}
