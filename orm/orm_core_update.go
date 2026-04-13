package orm

import (
	"context"
	"reflect"
	"time"

	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
	normalizeerr "github.com/siti-nabila/orm/pkg/normalize_err"
)

func (o *ORM) Update(ctx context.Context, v any, fields ...map[string]any) error {
	var (
		err               error
		start             = time.Now()
		updateQueryResult builder.UpdateQueryResult
		d                 dialect.Dialector
		mode              builder.DryRunMode
	)

	updateQueryResult, mode, d, err = o.prepareUpdate(v, fields...)
	if err != nil {
		return err
	}

	logCols := buildUpdateLogCols(updateQueryResult)

	defer func() {
		o.log(
			updateQueryResult.Query,
			d,
			logCols,
			updateQueryResult.Args,
			mode,
			start,
			err,
		)
	}()

	_, err = o.executor.ExecContext(ctx, updateQueryResult.Query, updateQueryResult.Args...)
	if err != nil {
		return normalizeerr.Normalize(d.Name(), err)
	}

	return nil
}

func (o *ORM) prepareUpdate(
	v any,
	fields ...map[string]any,
) (builder.UpdateQueryResult, builder.DryRunMode, dialect.Dialector, error) {
	var updateQueryResult builder.UpdateQueryResult

	rv := reflect.ValueOf(v)
	if !rv.IsValid() || rv.Kind() != reflect.Ptr || rv.IsNil() {
		return updateQueryResult, "", nil, dictionary.ErrMustBeStructPtr
	}

	if rv.Elem().Kind() != reflect.Struct {
		return updateQueryResult, "", nil, dictionary.ErrMustBeStructPtr
	}

	mode := o.placeholderMode()
	d := o.Dialect()

	var err error
	updateQueryResult, err = builder.BuildUpdateQuery(v, d, o.config, mode, fields...)
	if err != nil {
		return updateQueryResult, "", d, err
	}

	if updateQueryResult.Query == "" {
		return updateQueryResult, "", d, dictionary.ErrDBQueryEmpty
	}

	return updateQueryResult, builder.DryRunModeExec, d, nil
}
func buildUpdateLogCols(result builder.UpdateQueryResult) []mapper.ColumnMeta {
	cols := make([]mapper.ColumnMeta, 0, len(result.PlaceholderCols)+1)

	cols = append(cols, result.PlaceholderCols...)

	if result.PKColumn != nil {
		cols = append(cols, *result.PKColumn)
	}

	return cols
}
