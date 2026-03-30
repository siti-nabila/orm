package orm

import (
	"fmt"
	"reflect"

	"github.com/lib/pq"
)

func buildPostgresScanTarget(
	colName string,
	field reflect.Value,
) (any, *scanAssignment, bool, error) {
	// Prioritas: postgres array
	if field.Kind() == reflect.Slice {
		switch field.Type().Elem().Kind() {
		case reflect.String:
			return buildPostgresStringSliceScanTarget(colName, field)
		case reflect.Int64:
			return buildPostgresInt64SliceScanTarget(colName, field)
		case reflect.Int:
			return buildPostgresIntSliceScanTarget(colName, field)
		}
	}

	// Kalau tidak ada handling khusus, return handled=false
	return nil, nil, false, nil
}

func buildPostgresStringSliceScanTarget(
	colName string,
	field reflect.Value,
) (any, *scanAssignment, bool, error) {
	holder := new([]string)

	assignment := &scanAssignment{
		Field: field,
		AssignFunc: func() error {
			if holder == nil {
				field.Set(reflect.Zero(field.Type()))
				return nil
			}

			field.Set(reflect.ValueOf(*holder))
			return nil
		},
	}

	_ = colName
	return pq.Array(holder), assignment, true, nil
}

func buildPostgresInt64SliceScanTarget(
	colName string,
	field reflect.Value,
) (any, *scanAssignment, bool, error) {
	holder := new([]int64)

	assignment := &scanAssignment{
		Field: field,
		AssignFunc: func() error {
			if holder == nil {
				field.Set(reflect.Zero(field.Type()))
				return nil
			}

			field.Set(reflect.ValueOf(*holder))
			return nil
		},
	}

	_ = colName
	return pq.Array(holder), assignment, true, nil
}

func buildPostgresIntSliceScanTarget(
	colName string,
	field reflect.Value,
) (any, *scanAssignment, bool, error) {
	holder := new([]int64)

	assignment := &scanAssignment{
		Field: field,
		AssignFunc: func() error {
			if holder == nil {
				field.Set(reflect.Zero(field.Type()))
				return nil
			}

			result := reflect.MakeSlice(field.Type(), 0, len(*holder))
			for _, v := range *holder {
				if reflect.Zero(field.Type().Elem()).OverflowInt(v) {
					return fmt.Errorf("column %s value %d overflows int slice element", colName, v)
				}
				result = reflect.Append(result, reflect.ValueOf(int(v)).Convert(field.Type().Elem()))
			}

			field.Set(result)
			return nil
		},
	}

	return pq.Array(holder), assignment, true, nil
}
