package db

import (
	"database/sql"

	"github.com/siti-nabila/orm/internal/dialect"
)

func (tx *Tx) Exec(query string, args ...any) (sql.Result, error) {
	return tx.tx.Exec(query, args...)
}

func (tx *Tx) Query(query string, args ...any) (*sql.Rows, error) {
	return tx.tx.Query(query, args...)
}

func (tx *Tx) QueryRow(query string, args ...any) *sql.Row {
	return tx.tx.QueryRow(query, args...)
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
