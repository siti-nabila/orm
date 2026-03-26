package orm

import (
	"context"
	"time"

	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/db"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
	"github.com/siti-nabila/orm/pkg/helper"
	normalizeerr "github.com/siti-nabila/orm/pkg/normalize_err"
)

func (o *ORM) Create(ctx context.Context, v any) error {
	var (
		err          error
		start        = time.Now()
		query        string
		args         []any
		filteredCols []mapper.ColumnMeta
		d            dialect.Dialector
	)

	defer func() {
		o.log(query, d, filteredCols, args, start, err)
	}()
	meta, err := mapper.Parse(v, o.config.UseSnakeCase)
	if err != nil {
		return err
	}

	withReturningPrimary := true

	mode := o.placeholderMode()
	d = o.Dialect()
	insertQueryResult, err := builder.BuildInsertQuery(meta, d, o.config, mode, withReturningPrimary)
	if err != nil {
		return err
	}

	if insertQueryResult.Query == "" {
		return dictionary.ErrDBQueryEmpty
	}

	// if dialect supports returning primary key, then use query row and scan value to pkCol
	if insertQueryResult.PKColumn != nil && d.SupportReturning() {
		row := o.executor.QueryRowContext(ctx, insertQueryResult.Query, insertQueryResult.Args...)

		if err = row.Scan(insertQueryResult.PKColumn.FieldSrc.Addr().Interface()); err != nil {
			return normalizeerr.Normalize(d.Name(), err)
		}
		insertQueryResult.PKColumn.Value = insertQueryResult.PKColumn.FieldSrc.Interface()
		return nil
	}

	// if dialect does not support returning primary key, then use exec and check last insert id
	result, err := o.executor.ExecContext(ctx, insertQueryResult.Query, insertQueryResult.Args...)
	if err != nil {
		return normalizeerr.Normalize(d.Name(), err)
	}

	if insertQueryResult.PKColumn != nil && helper.IsIntKind(insertQueryResult.PKColumn.FieldSrc.Interface()) {
		lastID, err := result.LastInsertId()
		if err == nil {
			helper.SetAutoID(insertQueryResult.PKColumn.FieldSrc, lastID)
		}

	}

	return nil

}

func (o *ORM) Update(ctx context.Context, v any, fields ...map[string]any) (err error) {
	start := time.Now()

	d := o.Dialect()
	mode := o.placeholderMode()

	var res builder.UpdateQueryResult

	defer func() {

		o.log(
			res.Query,
			d,
			res.PlaceholderCols,
			res.Args,
			start,
			err,
		)
	}()
	res, err = builder.BuildUpdateQuery(v, d, o.config, mode, fields...)
	if err != nil {
		return err
	}

	if res.Query == "" {
		return dictionary.ErrDBQueryEmpty
	}

	_, err = o.executor.Exec(res.Query, res.Args...)
	if err != nil {
		return err
	}

	return nil
}

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
