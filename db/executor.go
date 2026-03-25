package db

import (
	"context"
	"database/sql"

	"github.com/siti-nabila/orm/dialect"
)

type (
	Executor interface {
		Exec(query string, args ...any) (sql.Result, error)
		Query(query string, args ...any) (*sql.Rows, error)
		QueryRow(query string, args ...any) *sql.Row

		ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
		QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
		QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row

		Dialect() dialect.Dialector
	}
)
