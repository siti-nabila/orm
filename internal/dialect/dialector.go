package dialect

type (
	Dialector interface {
		PlaceholderByNumber(n int) string
		PlaceholderByName(n string) string
		QuoteIdentifier(s string) string
		SupportReturning() bool
		Name() string
	}
)
