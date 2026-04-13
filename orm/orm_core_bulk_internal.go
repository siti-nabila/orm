package orm

import (
	"fmt"
	"reflect"

	"github.com/godev90/validator/faults"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
	"github.com/siti-nabila/orm/pkg/helper"
)

type (
	BulkInsertLayout struct {
		Table                string
		FilteredCols         []mapper.ColumnMeta
		PrimaryKeyColName    string
		PrimaryKeyColIndexes []int
		RowCount             int
	}
)

func validateBulkValues(values any) (reflect.Value, reflect.Type, bool, error) {
	rv := reflect.ValueOf(values)
	if !rv.IsValid() {
		return reflect.Value{}, nil, false, dictionary.ErrBulkInsertValueNil
	}

	if rv.Kind() != reflect.Ptr {
		return reflect.Value{}, nil, false, dictionary.ErrBulkInsertValueNotPointerSlice
	}

	if rv.IsNil() {
		return reflect.Value{}, nil, false, dictionary.ErrBulkInsertValueNil
	}

	sliceVal := rv.Elem()
	if sliceVal.Kind() != reflect.Slice {
		return reflect.Value{}, nil, false, dictionary.ErrBulkInsertValueNotPointerSlice
	}

	if sliceVal.Len() == 0 {
		return reflect.Value{}, nil, false, dictionary.ErrBulkInsertValueEmpty
	}

	elemType := sliceVal.Type().Elem()
	isPtrElem := false

	if elemType.Kind() == reflect.Ptr {
		isPtrElem = true
		elemType = elemType.Elem()
	}

	if elemType.Kind() != reflect.Struct {
		return reflect.Value{}, nil, false, dictionary.ErrBulkInsertValueSliceElementNotStruct
	}

	return sliceVal, elemType, isPtrElem, nil
}

func parseBulkMetas(values reflect.Value, isPtrElem bool, useSnake bool) ([]*mapper.Meta, error) {
	metas := make([]*mapper.Meta, 0, values.Len())
	errs := faults.Errors{}

	for i := 0; i < values.Len(); i++ {
		item := values.Index(i)

		var target any

		if isPtrElem {
			if item.IsNil() {
				errs[bulkRowKey(i)] = dictionary.ErrBulkInsertElemNil
				continue
			}
			target = item.Interface()
		} else {
			if !item.CanAddr() {
				errs[bulkRowKey(i)] = dictionary.ErrBulkInsertElemTypeMismatch
				continue
			}
			target = item.Addr().Interface()
		}

		meta, err := mapper.Parse(target, useSnake)
		if err != nil {
			errs[bulkRowKey(i)] = err
			continue
		}

		metas = append(metas, meta)
	}

	if len(errs) != 0 {
		return nil, errs
	}

	return metas, nil
}

func resolveBulkInsertLayout(metas []*mapper.Meta) (*BulkInsertLayout, error) {
	if len(metas) == 0 {
		return nil, dictionary.ErrBulkInsertEmptyMetas
	}

	first := metas[0]
	baseFilteredCols := filterBulkInsertColumns(first.Columns)
	if len(baseFilteredCols) == 0 {
		return nil, dictionary.ErrDBQueryEmpty
	}

	layout := &BulkInsertLayout{
		Table:                first.Table,
		FilteredCols:         baseFilteredCols,
		PrimaryKeyColName:    "",
		PrimaryKeyColIndexes: make([]int, 0),
		RowCount:             len(metas),
	}

	basePK := first.GetPrimaryKeyColumn()
	if basePK != nil {
		layout.PrimaryKeyColName = basePK.Name
	}

	for idx, col := range baseFilteredCols {
		if col.PrimaryKey {
			layout.PrimaryKeyColIndexes = append(layout.PrimaryKeyColIndexes, idx)
		}
	}

	for _, meta := range metas[1:] {
		if meta.Table != layout.Table {
			return nil, dictionary.ErrBulkInsertTableMismatch
		}

		pk := meta.GetPrimaryKeyColumn()

		switch {
		case basePK == nil && pk != nil:
			return nil, dictionary.ErrBulkInsertPrimaryKeyMismatch
		case basePK != nil && pk == nil:
			return nil, dictionary.ErrBulkInsertPrimaryKeyMismatch
		case basePK != nil && pk != nil && basePK.Name != pk.Name:
			return nil, dictionary.ErrBulkInsertPrimaryKeyMismatch
		}

		filteredCols := filterBulkInsertColumns(meta.Columns)

		if len(filteredCols) != len(layout.FilteredCols) {
			return nil, dictionary.ErrBulkInsertColumnCountMismatch
		}

		for colIdx := range layout.FilteredCols {
			if layout.FilteredCols[colIdx].Name != filteredCols[colIdx].Name {
				return nil, dictionary.ErrBulkInsertColumnMismatch
			}
		}
	}
	return layout, nil
}
func filterBulkInsertColumns(cols []mapper.ColumnMeta) []mapper.ColumnMeta {
	filtered := make([]mapper.ColumnMeta, 0, len(cols))

	for _, col := range cols {
		if col.PrimaryKey && helper.IsNilOrZeroValue(col.Value) {
			continue
		}

		filtered = append(filtered, col)
	}

	return filtered
}

func bulkRowKey(i int) string {
	return fmt.Sprintf("row %d", i)
}
