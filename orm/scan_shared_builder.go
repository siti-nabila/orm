package orm

import (
	"database/sql"
	"reflect"
	"strconv"
	"strings"

	"github.com/siti-nabila/orm/pkg/dictionary"
)

func buildSharedIntScanTarget(
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

func buildSharedUintScanTarget(
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

func buildSharedFloatScanTarget(
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

func buildSharedBytesScanTarget(
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

func buildSharedNullableStringScanTarget(
	colName string,
	field reflect.Value,
) (any, *scanAssignment, bool, error) {
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

	_ = colName
	return holder, assignment, true, nil
}

func buildSharedBoolScanTarget(
	colName string,
	field reflect.Value,
) (any, *scanAssignment, bool, error) {
	holder := new(sql.NullString)

	assignment := &scanAssignment{
		Field: field,
		AssignFunc: func() error {
			if !holder.Valid || holder.String == "" {
				field.SetBool(false)
				return nil
			}

			v := strings.TrimSpace(strings.ToLower(holder.String))
			field.SetBool(v == "1" || v == "true" || v == "y")
			return nil
		},
	}

	_ = colName
	return holder, assignment, true, nil
}
