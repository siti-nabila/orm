package dialect

import "fmt"

type (
	Oracle struct{}
)

func NewOracle() Oracle {
	return Oracle{}
}

func (d Oracle) PlaceholderByNumber(n int) string {
	return fmt.Sprintf(":%d", n)
}
func (d Oracle) PlaceholderByName(s string) string {
	return ":" + s
}

func (d Oracle) QuoteIdentifier(s string) string {
	return `"` + s + `"`
}

func (d Oracle) SupportReturning() bool {
	return true
}

func (d Oracle) Name() string {
	return "oracle"
}
