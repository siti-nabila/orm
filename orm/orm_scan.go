package orm

import (
	"context"
	"reflect"
	"time"

	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

func (o *ORM) ScanQuery(
	ctx context.Context,
	query string,
	args []any,
	selectedCols []mapper.ColumnMeta,
	dest any,
) error {
	start := time.Now()
	var (
		err  error
		mode builder.DryRunMode
	)

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
		mode = builder.DryRunModeQueryRow
	case reflect.Slice:
		mode = builder.DryRunModeQuery
	default:
		err = dictionary.ErrDBScanUnsupportedDest
		return err
	}

	defer func() {
		o.log(query, o.Dialect(), selectedCols, args, mode, start, err)
	}()

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
