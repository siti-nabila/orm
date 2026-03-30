package orm

import (
	"context"
	"reflect"
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
		err               error
		start             = time.Now()
		insertQueryResult builder.InsertQueryResult
		d                 dialect.Dialector
	)
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || rv.Kind() != reflect.Ptr || rv.IsNil() {
		return dictionary.ErrMustBeStructPtr
	}

	if rv.Elem().Kind() != reflect.Struct {
		return dictionary.ErrMustBeStructPtr
	}

	defer func() {
		o.log(insertQueryResult.Query, d, insertQueryResult.FilteredCols, insertQueryResult.Args, start, err)
	}()

	meta, err := mapper.Parse(v, o.config.UseSnakeCase)
	if err != nil {
		return err
	}

	withReturningPrimary := true

	mode := o.placeholderMode()
	d = o.Dialect()

	insertQueryResult, err = builder.BuildInsertQuery(meta, d, o.config, mode, withReturningPrimary)
	if err != nil {
		return err
	}
	if insertQueryResult.Query == "" {
		return dictionary.ErrDBQueryEmpty
	}

	if insertQueryResult.PKColumn != nil && d.SupportReturning() {
		row := o.executor.QueryRowContext(ctx, insertQueryResult.Query, insertQueryResult.Args...)

		if err = row.Scan(insertQueryResult.PKColumn.FieldSrc.Addr().Interface()); err != nil {
			return normalizeerr.Normalize(d.Name(), err)
		}
		insertQueryResult.PKColumn.Value = insertQueryResult.PKColumn.FieldSrc.Interface()
		return nil
	}

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
	var (
		start             = time.Now()
		updateQueryResult builder.UpdateQueryResult
	)

	rv := reflect.ValueOf(v)
	if !rv.IsValid() || rv.Kind() != reflect.Ptr || rv.IsNil() {
		return dictionary.ErrMustBeStructPtr
	}

	if rv.Elem().Kind() != reflect.Struct {
		return dictionary.ErrMustBeStructPtr
	}

	d := o.Dialect()
	mode := o.placeholderMode()

	defer func() {
		o.log(
			updateQueryResult.Query,
			d,
			updateQueryResult.PlaceholderCols,
			updateQueryResult.Args,
			start,
			err,
		)
	}()

	updateQueryResult, err = builder.BuildUpdateQuery(v, d, o.config, mode, fields...)
	if err != nil {
		return err
	}

	if updateQueryResult.Query == "" {
		return dictionary.ErrDBQueryEmpty
	}

	_, err = o.executor.ExecContext(ctx, updateQueryResult.Query, updateQueryResult.Args...)
	if err != nil {
		return err
	}

	return nil
}

func (o *ORM) ScanQuery(
	ctx context.Context,
	query string,
	args []any,
	selectedCols []mapper.ColumnMeta,
	dest any,
) error {
	start := time.Now()
	var err error
	defer func() {
		o.log(query, o.Dialect(), selectedCols, args, start, err)
	}()

	if dest == nil {
		err = dictionary.ErrDBScanNilDest
		return err
	}

	rv := reflect.ValueOf(dest)
	if !rv.IsValid() || rv.Kind() != reflect.Ptr || rv.IsNil() {
		err = dictionary.ErrDBScanNotPointerDest
		return err
	}

	elem := rv.Elem()

	switch elem.Kind() {
	case reflect.Struct:
		err = o.scanOne(ctx, query, args, dest)
		return err
	case reflect.Slice:
		err = o.scanMany(ctx, query, args, dest)
		return err
	default:
		err = dictionary.ErrDBScanUnsupportedDest
		return err
	}
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
