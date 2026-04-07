package orm

import (
	"database/sql"
	"reflect"

	"github.com/siti-nabila/orm/pkg/dictionary"
)

func buildUintScanTarget(
	colName string,
	field reflect.Value,
) (any, *scanAssignment, error) {
	holder := new(sql.NullInt64)

	assignment := &scanAssignment{
		Field: field,
		AssignFunc: func() error {
			if !holder.Valid {
				field.SetUint(0)
				return nil
			}

			if holder.Int64 < 0 {
				return dictionary.ErrNegativeValueUintError(colName, holder.Int64)
				// fmt.Errorf("column %s has negative value %d for unsigned field", colName, holder.Int64)
			}

			v := uint64(holder.Int64)
			if field.OverflowUint(v) {
				return dictionary.ErrColOverflowError(colName, holder.Int64)
				// fmt.Errorf("column %s value %d overflows unsigned field", colName, holder.Int64)
			}

			field.SetUint(v)
			return nil
		},
	}

	return holder, assignment, nil
}
