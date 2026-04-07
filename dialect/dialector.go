package dialect

const (
	DialectPostgres DialectType = "postgres"
	DialectMySQL    DialectType = "mysql"
	DialectOracle   DialectType = "oracle"
)

type (
	DialectType string
	Dialector   interface {
		PlaceholderByNumber(n int) string
		PlaceholderByName(n string) string
		QuoteIdentifier(s string) string
		SupportReturning() bool
		Name() string
		Type() DialectType
	}
)
