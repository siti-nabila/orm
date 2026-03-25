package db

import (
	"database/sql"

	"github.com/siti-nabila/orm/dialect"
)

type (
	Executor interface {
		Exec(query string, args ...any) (sql.Result, error)
		Query(query string, args ...any) (*sql.Rows, error)
		QueryRow(query string, args ...any) *sql.Row
		Dialect() dialect.Dialector
	}
)
