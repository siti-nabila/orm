package helper

import (
	"reflect"
	"strings"

	"github.com/siti-nabila/orm/pkg/dictionary"
)

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

func IndirectType(t reflect.Type) reflect.Type {
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func IndirectValue(v reflect.Value) reflect.Value {
	for v.IsValid() && v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

func IsAllowedPointerStruct(v any) error {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || rv.Kind() != reflect.Ptr || rv.IsNil() {
		return dictionary.ErrMustBeStructPtr
	}
	if rv.Elem().Kind() != reflect.Struct {
		return dictionary.ErrMustBeStructPtr
	}
	return nil
}

func IsExpandableSliceArg(v any) bool {
	if v == nil {
		return false
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return false
	}

	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return false
	}

	// []byte jangan dianggap IN list
	if rv.Type().Elem().Kind() == reflect.Uint8 {
		return false
	}

	return true
}

func IsRawSelectExpr(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}

	upper := strings.ToUpper(s)

	if strings.Contains(s, " ") ||
		strings.Contains(s, ".") ||
		strings.Contains(s, "(") ||
		strings.Contains(s, ")") ||
		strings.Contains(s, "*") ||
		strings.Contains(upper, " AS ") {
		return true
	}

	return false
}

func IsNilOrZeroValue(v any) bool {
	if v == nil {
		return true
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return true
	}

	for rv.Kind() == reflect.Interface || rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return true
		}
		rv = rv.Elem()
	}

	return rv.IsZero()
}

Beberapa catatan penting untuk file ini:

validateBulkValues