package dialect

import (
	"context"
	"fmt"

	"github.com/siti-nabila/orm/lock"
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
	return s
}

func (d Postgres) SupportReturning() bool {
	return true
}

func (d Postgres) Name() string {
	return "postgres"
}

func (d Postgres) Type() DialectType {
	return DialectPostgres
}

func (d Postgres) TryLockQuery(ctx context.Context, key string) (query string, args []any, err error) {
	hash := lock.NewHash64(key)
	return `SELECT pg_try_advisory_xact_lock($1)`, []any{hash}, nil
}

func (d Postgres) ReleaseLockQuery(ctx context.Context, key string) (query string, args []any, needed bool, err error) {
	return "", nil, false, nil
}
