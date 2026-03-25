package db

import (
	"database/sql"

	"github.com/siti-nabila/orm/dialect"
)

type (
	DB struct {
		conn    *sql.DB
		dialect dialect.Dialector
	}
)

func New(conn *sql.DB, dialect dialect.Dialector) *DB {
	return &DB{
		conn:    conn,
		dialect: dialect,
	}
}
