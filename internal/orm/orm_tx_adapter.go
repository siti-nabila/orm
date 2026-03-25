package orm

import (
	"context"
	"database/sql"

	"github.com/siti-nabila/orm/internal/config"
	"github.com/siti-nabila/orm/internal/db"
	"github.com/siti-nabila/orm/internal/dialect"
)

type (
	SqlTransactionAdapter struct {
		ctx context.Context
		tx  *sql.Tx
		orm *ORM
	}
)

func NewSqlTransactionAdapter(
	ctx context.Context,
	tx *sql.Tx,
	d dialect.Dialector,
	cfg config.Config,
) *SqlTransactionAdapter {
	exec := db.NewTx(tx, d)
	o := New(exec, cfg)

	return &SqlTransactionAdapter{
		ctx: ctx,
		tx:  tx,
		orm: o,
	}
}
