package db

import (
	"database/sql"

	"github.com/siti-nabila/orm/internal/dialect"
)

type (
	Tx struct {
		tx      *sql.Tx
		dialect dialect.Dialector
	}
)

func NewTx(tx *sql.Tx, dialect dialect.Dialector) *Tx {
	return &Tx{
		tx:      tx,
		dialect: dialect,
	}
}
