package orm

import (
	"time"

	"github.com/siti-nabila/orm/builder"
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

func (o *ORM) DryRunCreate(v any) (builder.DryRunResult, error) {
	insertQueryResult, mode, d, err := o.prepareCreate(v)
	if err != nil {
		return builder.DryRunResult{}, err
	}

	res := builder.DryRunResult{
		Query: insertQueryResult.Query,
		Args:  insertQueryResult.Args,
		Mode:  mode,
	}

	o.logDryRun(
		res.Query,
		d,
		insertQueryResult.FilteredCols,
		res.Args,
		res.Mode,
	)

	return res, nil
}

func (o *ORM) DryRunUpdate(v any, fields ...map[string]any) (builder.DryRunResult, error) {
	updateQueryResult, mode, d, err := o.prepareUpdate(v, fields...)
	if err != nil {
		return builder.DryRunResult{}, err
	}

	logCols := buildUpdateLogCols(updateQueryResult)

	res := builder.DryRunResult{
		Query: updateQueryResult.Query,
		Args:  updateQueryResult.Args,
		Mode:  mode,
	}

	o.logDryRun(
		res.Query,
		d,
		logCols,
		res.Args,
		res.Mode,
	)

	return res, nil
}

func (o *ORM) log(
	query string,
	d dialect.Dialector,
	cols []mapper.ColumnMeta,
	args []any,
	mode builder.DryRunMode,
	start time.Time,
	err error,
) {
	if o.logger == nil || !o.debug {
		return
	}

	o.logger.Log(
		query,
		d,
		cols,
		args,
		mode.String(),
		time.Since(start),
		err,
	)
}
func (o *ORM) logDryRun(
	query string,
	d dialect.Dialector,
	cols []mapper.ColumnMeta,
	args []any,
	mode builder.DryRunMode,
) {
	if o.logger == nil || !o.config.LogDryRunQuery {
		return
	}

	o.logger.LogDryRun(
		query,
		d,
		cols,
		args,
		mode.String(),
	)
}

func (o *ORM) shouldLogLockQuery() bool {
	return o != nil && o.config.LogLockQuery
}
