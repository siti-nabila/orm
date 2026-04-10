package orm

import (
	"context"
	"database/sql"
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
		mode              builder.DryRunMode
	)

	insertQueryResult, mode, d, err = o.prepareCreate(v)
	if err != nil {
		return err
	}

	defer func() {
		o.log(
			insertQueryResult.Query,
			d,
			insertQueryResult.FilteredCols,
			insertQueryResult.Args,
			mode,
			start,
			err,
		)
	}()

	switch mode {
	case builder.DryRunModeQueryRow:
		row := o.executor.QueryRowContext(ctx, insertQueryResult.Query, insertQueryResult.Args...)
		if err = row.Scan(insertQueryResult.PKColumn.FieldSrc.Addr().Interface()); err != nil {
			return normalizeerr.Normalize(d.Name(), err)
		}

		insertQueryResult.PKColumn.Value = insertQueryResult.PKColumn.FieldSrc.Interface()
		return nil

	case builder.DryRunModeExec:
		var result sql.Result
		result, err = o.executor.ExecContext(ctx, insertQueryResult.Query, insertQueryResult.Args...)
		if err != nil {
			return normalizeerr.Normalize(d.Name(), err)
		}

		if insertQueryResult.PKColumn != nil && helper.IsIntKind(insertQueryResult.PKColumn.FieldSrc.Interface()) {
			lastID, lastIDErr := result.LastInsertId()
			if lastIDErr == nil {
				helper.SetAutoID(insertQueryResult.PKColumn.FieldSrc, lastID)
			}
		}
		return nil

	default:
		return dictionary.ErrDBQueryEmpty
	}
}
func (o *ORM) prepareCreate(v any) (builder.InsertQueryResult, builder.DryRunMode, dialect.Dialector, error) {
	var insertQueryResult builder.InsertQueryResult

	rv := reflect.ValueOf(v)
	if !rv.IsValid() || rv.Kind() != reflect.Ptr || rv.IsNil() {
		return insertQueryResult, "", nil, dictionary.ErrMustBeStructPtr
	}

	if rv.Elem().Kind() != reflect.Struct {
		return insertQueryResult, "", nil, dictionary.ErrMustBeStructPtr
	}

	meta, err := mapper.Parse(v, o.config.UseSnakeCase)
	if err != nil {
		return insertQueryResult, "", nil, err
	}

	withReturningPrimary := true
	mode := o.placeholderMode()
	d := o.Dialect()

	insertQueryResult, err = builder.BuildInsertQuery(meta, d, o.config, mode, withReturningPrimary)
	if err != nil {
		return insertQueryResult, "", d, err
	}
	if insertQueryResult.Query == "" {
		return insertQueryResult, "", d, dictionary.ErrDBQueryEmpty
	}

	execMode := resolveCreateMode(insertQueryResult, d)

	return insertQueryResult, execMode, d, nil
}

func resolveCreateMode(result builder.InsertQueryResult, d dialect.Dialector) builder.DryRunMode {
	if result.PKColumn != nil && d.SupportReturning() {
		return builder.DryRunModeQueryRow
	}
	return builder.DryRunModeExec
}
