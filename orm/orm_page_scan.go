package orm

import (
	"context"
	"reflect"
	"strings"
	"time"

	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
	normalizeerr "github.com/siti-nabila/orm/pkg/normalize_err"
)

func (o *ORM) ScanPageQuery(
	ctx context.Context,
	query string,
	args []any,
	selectedCols []mapper.ColumnMeta,
	dest any,
	totalColumn string,
) (total *int64, err error) {
	start := time.Now()
	defer func() {
		o.log(query, o.Dialect(), selectedCols, args, builder.DryRunModeQuery, start, err)
	}()

	rows, err := o.executor.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, normalizeerr.Normalize(o.Dialect().Name(), err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	modelColumns := make([]string, 0, len(columns)-1)
	totalIndex := -1
	for i, column := range columns {
		if strings.EqualFold(column, totalColumn) {
			totalIndex = i
			continue
		}
		modelColumns = append(modelColumns, column)
	}
	if totalIndex < 0 {
		return nil, dictionary.ErrPaginationTotalColumnNotFound
	}

	rv := reflect.ValueOf(dest)
	destSlice := rv.Elem()
	elemType := destSlice.Type().Elem()
	resultSlice := reflect.MakeSlice(destSlice.Type(), 0, 8)

	template := reflect.New(elemType)
	templateMeta, err := mapper.Parse(template.Interface(), o.config.UseSnakeCase)
	if err != nil {
		return nil, err
	}
	scanIndexes, err := prepareScanIndexes(templateMeta, modelColumns)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		elemPtr := reflect.New(elemType)
		meta, err := mapper.Parse(elemPtr.Interface(), o.config.UseSnakeCase)
		if err != nil {
			return nil, err
		}
		modelTargets, assignments, err := prepareScanTargets(meta, scanIndexes, o.Dialect())
		if err != nil {
			return nil, normalizeerr.Normalize(o.Dialect().Name(), err)
		}

		var rowTotal int64
		targets := make([]any, len(columns))
		modelTargetIndex := 0
		for i := range columns {
			if i == totalIndex {
				targets[i] = &rowTotal
				continue
			}
			targets[i] = modelTargets[modelTargetIndex]
			modelTargetIndex++
		}

		if err := rows.Scan(targets...); err != nil {
			return nil, normalizeerr.Normalize(o.Dialect().Name(), err)
		}
		if err := applyScanAssignments(assignments); err != nil {
			return nil, err
		}
		if total == nil && totalIndex >= 0 {
			total = new(int64)
			*total = rowTotal
		}
		resultSlice = reflect.Append(resultSlice, elemPtr.Elem())
	}

	if err := rows.Err(); err != nil {
		return nil, normalizeerr.Normalize(o.Dialect().Name(), err)
	}

	destSlice.Set(resultSlice)
	return total, nil
}
