package db

import (
	"context"
	"database/sql"

	"github.com/siti-nabila/orm/dialect"
)

func (tx *Tx) Exec(query string, args ...any) (sql.Result, error) {
	return tx.tx.Exec(query, normalizeArgs(tx.dialect, args)...)
}

func (tx *Tx) Query(query string, args ...any) (*sql.Rows, error) {
	return tx.tx.Query(query, normalizeArgs(tx.dialect, args)...)
}

func (tx *Tx) QueryRow(query string, args ...any) *sql.Row {
	return tx.tx.QueryRow(query, normalizeArgs(tx.dialect, args)...)
}

func (tx *Tx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return tx.tx.QueryRowContext(ctx, query, normalizeArgs(tx.dialect, args)...)
}

func (tx *Tx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return tx.tx.QueryContext(ctx, query, normalizeArgs(tx.dialect, args)...)

}
func (tx *Tx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return tx.tx.ExecContext(ctx, query, normalizeArgs(tx.dialect, args)...)
}

func (tx *Tx) Commit() error {
	return tx.tx.Commit()
}

func (tx *Tx) Rollback() error {
	return tx.tx.Rollback()
}

func (tx *Tx) Dialect() dialect.Dialector {
	return tx.dialect
}
