package dialect

import "github.com/siti-nabila/orm/pkg/dictionary"

type (
	Mysql struct{}
)

func NewMysql() Mysql {
	return Mysql{}
}

func (d Mysql) PlaceholderByNumber(n int) string {
	return "?"
}

func (d Mysql) PlaceholderByName(n string) string {
	panic(dictionary.UnsupportedTypeError(d.Name()))
}

func (d Mysql) QuoteIdentifier(s string) string {
	return "`" + s + "`"
}

func (d Mysql) SupportReturning() bool {
	return false
}

func (d Mysql) Name() string {
	return "mysql"
}
