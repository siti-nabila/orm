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
	query, args, pkCol, filteredCols := builder.BuildInsertQuery(meta, d, o.config, mode, withReturningPrimary)

	if query == "" {
		return dictionary.ErrDBQueryEmpty
	}

	// if dialect supports returning primary key, then use query row and scan value to pkCol
	if pkCol != nil && d.SupportReturning() {
		row := o.executor.QueryRowContext(ctx, query, args...)

		if err = row.Scan(pkCol.FieldSrc.Addr().Interface()); err != nil {
			return normalizeerr.Normalize(d.Name(), err)
		}
		pkCol.Value = pkCol.FieldSrc.Interface()
		return nil
	}

	// if dialect does not support returning primary key, then use exec and check last insert id
	result, err := o.executor.ExecContext(ctx, query, args...)
	if err != nil {
		return normalizeerr.Normalize(d.Name(), err)
	}

	if pkCol != nil && helper.IsIntKind(pkCol.FieldSrc.Interface()) {
		lastID, err := result.LastInsertId()
		if err == nil {
			helper.SetAutoID(pkCol.FieldSrc, lastID)
		}

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

func (o *ORM) log(query string, d dialect.Dialector, cols []mapper.ColumnMeta, args []any, start time.Time, err error) {
	if o.logger == nil || !o.debug || query == "" {
		return
	}
	duration := time.Since(start)
	o.logger.Log(query, d, cols, args, duration, err)
}
