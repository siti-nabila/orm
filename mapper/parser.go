package mapper

import "reflect"

func Parse(v any, useSnake bool) (*Meta, error) {
	val := reflect.ValueOf(v)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()

	meta := &Meta{}

	meta.Table = getTableName(
		v,
		typ,
		useSnake,
	)

	for i := 0; i < val.NumField(); i++ {

		fType := typ.Field(i)
		fVal := val.Field(i)

		col, ok := parseSQLTag(
			fType,
			useSnake,
		)

		if !ok {
			continue
		}

		col.Value = fVal.Interface()
		col.FieldSrc = fVal

		meta.Columns = append(
			meta.Columns,
			col,
		)
	}

	return meta, nil
}
