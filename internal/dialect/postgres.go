package dialect

import (
	"fmt"

	"github.com/siti-nabila/orm/pkg/dictionary"
)

type (
	Postgres struct{}
)

func NewPostgres() Postgres {
	return Postgres{}
}

func (d Postgres) PlaceholderByNumber(n int) string {
	return fmt.Sprintf("$%d", n)
}

func (d Postgres) PlaceholderByName(n string) string {
	panic(dictionary.UnsupportedTypeError(d.Name()))
}

func (d Postgres) QuoteIdentifier(s string) string {
	return `"` + s + `"`
}

func (d Postgres) SupportReturning() bool {
	return true
}

func (d Postgres) Name() string {
	return "postgres"
}
