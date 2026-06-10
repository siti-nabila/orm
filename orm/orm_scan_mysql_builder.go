package orm

import (
	"reflect"
)

func buildMySQLScanTarget(
	colName string,
	field reflect.Value,
) (any, *scanAssignment, bool, error) {

	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return buildSharedIntScanTarget(colName, field)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return buildSharedUintScanTarget(colName, field)

	case reflect.Float32, reflect.Float64:
		return buildSharedFloatScanTarget(colName, field)

	case reflect.Bool:
		return buildSharedBoolScanTarget(colName, field)

	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.Uint8 {
			return buildSharedBytesScanTarget(colName, field)
		}
	}

	// *string
	if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.String {
		return buildSharedNullableStringScanTarget(colName, field)
	}

	return nil, nil, false, nil
}
