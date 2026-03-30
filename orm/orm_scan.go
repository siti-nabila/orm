package orm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
	normalizeerr "github.com/siti-nabila/orm/pkg/normalize_err"
)

func (o *ORM) scanOne(
	ctx context.Context,
	query string,
	args []any,
	dest any,
) error {
	rows, err := o.executor.QueryContext(ctx, query, args...)
	if err != nil {
		return normalizeerr.Normalize(o.Dialect().Name(), err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return normalizeerr.Normalize(o.Dialect().Name(), err)
		}
		return normalizeerr.Normalize(o.Dialect().Name(), sql.ErrNoRows)
	}

	meta, err := mapper.Parse(dest, o.config.UseSnakeCase)
	if err != nil {
		return err
	}

	if err := scanCurrentRowIntoStruct(rows, cols, meta, o.Dialect()); err != nil {
		return normalizeerr.Normalize(o.Dialect().Name(), err)
	}

	if err := rows.Err(); err != nil {
		return normalizeerr.Normalize(o.Dialect().Name(), err)
	}

	return nil
}

func (o *ORM) scanMany(
	ctx context.Context,
	query string,
	args []any,
	dest any,
) error {
	rows, err := o.executor.QueryContext(ctx, query, args...)
	if err != nil {
		return normalizeerr.Normalize(o.Dialect().Name(), err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	rv := reflect.ValueOf(dest)
	destSlice := rv.Elem()
	elemType := destSlice.Type().Elem()

	if elemType.Kind() != reflect.Struct {
		return dictionary.ErrDBScanMustBeSliceStruct
	}

	resultSlice := reflect.MakeSlice(destSlice.Type(), 0, 0)

	for rows.Next() {
		elemPtr := reflect.New(elemType)

		meta, err := mapper.Parse(elemPtr.Interface(), o.config.UseSnakeCase)
		if err != nil {
			return err
		}

		if err := scanCurrentRowIntoStruct(rows, cols, meta, o.Dialect()); err != nil {
			return normalizeerr.Normalize(o.Dialect().Name(), err)
		}

		resultSlice = reflect.Append(resultSlice, elemPtr.Elem())
	}

	if err := rows.Err(); err != nil {
		return normalizeerr.Normalize(o.Dialect().Name(), err)
	}

	destSlice.Set(resultSlice)
	return nil
}

func scanCurrentRowIntoStruct(
	rows *sql.Rows,
	cols []string,
	meta *mapper.Meta,
	d dialect.Dialector,
) error {
	scanIndexes, err := prepareScanIndexes(meta, cols)
	if err != nil {
		return err
	}

	scanTargets, assignments, err := prepareScanTargets(meta, scanIndexes, d)
	if err != nil {
		return err
	}

	if err := rows.Scan(scanTargets...); err != nil {
		return err
	}

	if err := applyScanAssignments(assignments); err != nil {
		return err
	}

	return nil
}
func buildScanIndexesFromColumns(meta *mapper.Meta, cols []string) ([]int, error) {
	if meta == nil {
		return nil, dictionary.ErrDBScanMetaNil
	}

	if len(cols) == 0 {
		return nil, dictionary.ErrDBQueryEmpty
	}

	indexes := make([]int, 0, len(cols))

	for _, colName := range cols {
		idx, ok := meta.ColumnIndex[colName]
		if !ok {
			return nil, fmt.Errorf("column %s not found in destination struct", colName)
		}

		indexes = append(indexes, idx)
	}

	return indexes, nil
}

func prepareScanTargets(meta *mapper.Meta, scanIndexes []int, d dialect.Dialector) ([]any, []scanAssignment, error) {
	return bindScanTargetsByIndexes(meta, scanIndexes, d)
}

func prepareScanIndexes(meta *mapper.Meta, cols []string) ([]int, error) {
	return buildScanIndexesFromColumns(meta, cols)
}

func bindScanTargetsByIndexes(
	meta *mapper.Meta,
	indexes []int,
	d dialect.Dialector,
) ([]any, []scanAssignment, error) {
	if meta == nil {
		return nil, nil, dictionary.ErrDBScanMetaNil
	}

	targets := make([]any, 0, len(indexes))
	assignments := make([]scanAssignment, 0)

	for _, idx := range indexes {
		if idx < 0 || idx >= len(meta.Columns) {
			return nil, nil, dictionary.ErrInvalidValue
		}
		col := meta.Columns[idx]
		field := col.FieldSrc

		if !field.IsValid() || !field.CanAddr() {
			return nil, nil, fmt.Errorf("field for column %s is not addressable", col.Name)
		}

		target, assignment, err := buildScanTargetForField(d, col.Name, field)
		if err != nil {
			return nil, nil, err
		}

		targets = append(targets, target)
		if assignment != nil {
			assignments = append(assignments, *assignment)
		}
	}

	return targets, assignments, nil
}

func buildScanTargetForField(
	d dialect.Dialector,
	colName string,
	field reflect.Value,
) (any, *scanAssignment, error) {

	if !field.IsValid() || !field.CanAddr() {
		return nil, nil, fmt.Errorf("field for column %s is not addressable", colName)
	}

	dialectName := strings.ToLower(strings.TrimSpace(d.Name()))
	isPostgres := ""
	if strings.Contains(dialectName, "postgres") {
		isPostgres = "postgres"
	}
	// Handle scan based on driver-specific quirks
	switch dialectName {
	case isPostgres:
		target, assignment, handled, err := buildPostgresScanTarget(colName, field)
		if err != nil {
			return nil, nil, err
		}
		if handled {
			return target, assignment, nil
		}
	case "mysql":
		target, assignment, handled, err := buildMySQLScanTarget(colName, field)
		if err != nil {
			return nil, nil, err
		}
		if handled {
			return target, assignment, nil
		}

	case "oracle":
		target, assignment, handled, err := buildOracleScanTarget(colName, field)
		if err != nil {
			return nil, nil, err
		}
		if handled {
			return target, assignment, nil
		}
	}
	// generic priority: sql.Scanner
	if scanner, ok := field.Addr().Interface().(sql.Scanner); ok {
		return scanner, nil, nil
	}

	switch field.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return buildUintScanTarget(colName, field)
	default:
		return field.Addr().Interface(), nil, nil
	}
}

func buildMySQLScanTarget(
	colName string,
	field reflect.Value,
) (any, *scanAssignment, bool, error) {
	_ = colName
	_ = field
	// fokus untuk postgres dulu, nanti handling MySQL scan yang tipe data khusus bisa dsini
	return nil, nil, false, nil
}

func buildOracleScanTarget(
	colName string,
	field reflect.Value,
) (any, *scanAssignment, bool, error) {
	_ = colName
	_ = field
	// fokus untuk postgres dulu, nanti handling MySQL scan yang tipe data khusus bisa dsini

	return nil, nil, false, nil
}

func applyScanAssignments(assignments []scanAssignment) error {
	for _, a := range assignments {
		if a.AssignFunc == nil {
			continue
		}
		if err := a.AssignFunc(); err != nil {
			return err
		}
	}
	return nil
}
