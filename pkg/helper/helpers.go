package helper

import "reflect"

func IsZero(v any) bool {

	if v == nil {
		return true
	}

	val := reflect.ValueOf(v)

	switch val.Kind() {

	case reflect.Ptr, reflect.Interface:
		return val.IsNil()

	default:
		z := reflect.Zero(val.Type()).Interface()
		return reflect.DeepEqual(v, z)
	}
}

func IsIntKind(v any) bool {

	val := reflect.ValueOf(v)

	switch val.Kind() {
	case
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64:
		return true
	}

	return false
}

func SetAutoID(field reflect.Value, id int64) {
	// handle pointer
	if field.Kind() == reflect.Ptr {

		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}

		field = field.Elem()
	}

	switch field.Kind() {

	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64:

		field.SetInt(id)

	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64:

		field.SetUint(uint64(id))

	default:
		// ignore, bukan numeric
	}
}
