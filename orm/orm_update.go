package orm

import (
	"context"
	"reflect"
	"time"

	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

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
