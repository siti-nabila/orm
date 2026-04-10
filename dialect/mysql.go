package dialect

import (
	"context"

	"github.com/siti-nabila/orm/pkg/dictionary"
)

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

func (d Mysql) Type() DialectType {
	return DialectMySQL
}

func (d Mysql) TryLockQuery(ctx context.Context, key string) (query string, args []any, err error) {
	return `SELECT GET_LOCK(?, 0)`, []any{key}, nil
}

func (d Mysql) ReleaseLockQuery(ctx context.Context, key string) (query string, args []any, needed bool, err error) {
	return `SELECT RELEASE_LOCK(?)`, []any{key}, true, nil
}
