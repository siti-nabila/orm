package orm

import (
	"context"
	"database/sql"
	"reflect"

	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/db"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/pkg/logger"
)

type (
	SqlQueryAdapter struct {
		ctx context.Context
		db  *sql.DB
		orm *ORM
	}
	scanAssignment struct {
		Field      reflect.Value
		AssignFunc func() error
	}
)

func NewSqlQueryAdapter(
	ctx context.Context,
	conn *sql.DB,
	dialector dialect.Dialector,
	cfg config.Config,
) *SqlQueryAdapter {
	exec := db.New(conn, dialector)
	o := New(exec, cfg)

	o.SetLogger(logger.DefaultLogger{}, cfg.EnableDebug)
	return &SqlQueryAdapter{
		ctx: ctx,
		db:  conn,
		orm: o,
	}
}
