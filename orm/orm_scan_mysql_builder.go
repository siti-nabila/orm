package orm

import (
	"database/sql"
	"reflect"

	"github.com/siti-nabila/orm/pkg/dictionary"
)

func buildMySQLScanTarget(
	colName string,
	field reflect.Value,
) (any, *scanAssignment, bool, error) {

	// []byte
	if field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.Uint8 {
		holder := new([]byte)

		assignment := &scanAssignment{
			Field: field,
			AssignFunc: func() error {
				if holder == nil || *holder == nil {
					field.Set(reflect.Zero(field.Type()))
					return nil
				}

				switch v := any(*holder).(type) {
				case []byte:
					cloned := append([]byte(nil), v...)
					field.Set(reflect.ValueOf(cloned))
					return nil

				default:
					return dictionary.ErrScanTypeMismatch(colName, v)
				}
			},
		}

		return holder, assignment, true, nil
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
