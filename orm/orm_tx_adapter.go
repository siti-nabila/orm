package orm

import (
	"context"
	"database/sql"

	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/db"
	"github.com/siti-nabila/orm/dialect"
)

type (
	SqlTransactionAdapter struct {
		ctx          context.Context
		tx           *sql.Tx
		orm          *ORM
		acquiredLock map[string]struct{}
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
		ctx:          ctx,
		tx:           tx,
		orm:          o,
		acquiredLock: make(map[string]struct{}),
	}
}
