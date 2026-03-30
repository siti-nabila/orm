package orm

import (
	"database/sql"
	"fmt"
	"reflect"
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
				return fmt.Errorf("column %s has negative value %d for unsigned field", colName, holder.Int64)
			}

			v := uint64(holder.Int64)
			if field.OverflowUint(v) {
				return fmt.Errorf("column %s value %d overflows unsigned field", colName, holder.Int64)
			}

			field.SetUint(v)
			return nil
		},
	}

	return holder, assignment, nil
}
