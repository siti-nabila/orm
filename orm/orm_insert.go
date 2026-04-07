package orm

import (
	"context"
	"reflect"
	"time"

	"github.com/siti-nabila/orm/builder"
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
