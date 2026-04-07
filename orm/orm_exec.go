package orm

import (
	"time"

	"github.com/siti-nabila/orm/db"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

func (o *ORM) Begin() (*ORM, error) {
	db, ok := o.executor.(*db.DB)
	if !ok {
		return nil, dictionary.ErrDBConn
	}
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	return &ORM{
		executor: tx,
		config:   o.config,
	}, nil
}

func (o *ORM) Commit() error {
	tx, ok := o.executor.(*db.Tx)
	if !ok {
		return dictionary.ErrDBConn
	}
	return tx.Commit()
}

func (o *ORM) Rollback() error {
	tx, ok := o.executor.(*db.Tx)
	if !ok {
		return dictionary.ErrDBConn
	}
	return tx.Rollback()
}

func (o *ORM) Dialect() dialect.Dialector {
	return o.executor.Dialect()
}

func (o *ORM) log(
	query string,
	d dialect.Dialector,
	cols []mapper.ColumnMeta,
	args []any,
	start time.Time,
	err error,
) {
	if o.logger == nil || !o.debug || query == "" {
		return
	}

	duration := time.Since(start)
	o.logger.Log(query, d, cols, args, duration, err)
}
