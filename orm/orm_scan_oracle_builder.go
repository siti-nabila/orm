package orm

import (
	"database/sql"
	"reflect"
	"strconv"

	"github.com/siti-nabila/orm/pkg/dictionary"
)

func buildOracleScanTarget(
	colName string,
	field reflect.Value,
) (any, *scanAssignment, bool, error) {
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return buildOracleIntScanTarget(colName, field)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return buildOracleUintScanTarget(colName, field)

	case reflect.Float32, reflect.Float64:
		return buildOracleFloatScanTarget(colName, field)

	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.Uint8 {
			return buildOracleBytesScanTarget(colName, field)
		}
	}

	// *string
	if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.String {
		holder := new(sql.NullString)

		assignment := &scanAssignment{
			Field: field,
			AssignFunc: func() error {
				if !holder.Valid {
					field.Set(reflect.Zero(field.Type()))
					return nil
				}

				v := holder.String
				field.Set(reflect.ValueOf(&v))
				return nil
			},
		}

		return holder, assignment, true, nil
	}

	return nil, nil, false, nil
}

func buildOracleIntScanTarget(
	colName string,
	field reflect.Value,
) (any, *scanAssignment, bool, error) {

	holder := new(sql.NullString)

	assignment := &scanAssignment{
		Field: field,
		AssignFunc: func() error {
			if !holder.Valid || holder.String == "" {
				field.SetInt(0)
				return nil
			}

			v, err := strconv.ParseInt(holder.String, 10, 64)
			if err != nil {
				return dictionary.ErrScanTypeIntMismatch(colName, holder.String)
			}

			if field.OverflowInt(v) {
				return dictionary.ErrColOverflowError(colName, v)
			}

			field.SetInt(v)
			return nil
		},
	}

	return holder, assignment, true, nil
}
func buildOracleUintScanTarget(
	colName string,
	field reflect.Value,
) (any, *scanAssignment, bool, error) {
	holder := new(sql.NullString)

	assignment := &scanAssignment{
		Field: field,
		AssignFunc: func() error {
			if !holder.Valid || holder.String == "" {
				field.SetUint(0)
				return nil
			}

			v, err := strconv.ParseUint(holder.String, 10, 64)
			if err != nil {
				return dictionary.ErrInvalidValue
			}
			if field.OverflowUint(v) {
				return dictionary.ErrColOverflowError(colName, v)
			}

			field.SetUint(v)
			return nil
		},
	}

	return holder, assignment, true, nil
}

func buildOracleFloatScanTarget(
	colName string,
	field reflect.Value,
) (any, *scanAssignment, bool, error) {
	holder := new(sql.NullString)

	assignment := &scanAssignment{
		Field: field,
		AssignFunc: func() error {
			if !holder.Valid || holder.String == "" {
				field.SetFloat(0)
				return nil
			}

			v, err := strconv.ParseFloat(holder.String, 64)
			if err != nil {
				return dictionary.ErrInvalidValue
			}
			if field.OverflowFloat(v) {
				return dictionary.ErrColOverflowError(colName, v)
			}

			field.SetFloat(v)
			return nil
		},
	}

	return holder, assignment, true, nil
}

func buildOracleBytesScanTarget(
	colName string,
	field reflect.Value,
) (any, *scanAssignment, bool, error) {
	holder := new([]byte)

	assignment := &scanAssignment{
		Field: field,
		AssignFunc: func() error {
			if holder == nil || *holder == nil {
				field.Set(reflect.Zero(field.Type()))
				return nil
			}

			cloned := append([]byte(nil), (*holder)...)
			field.Set(reflect.ValueOf(cloned))
			return nil
		},
	}

	_ = colName
	return holder, assignment, true, nil
}
