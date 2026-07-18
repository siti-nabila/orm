package db

import (
	"database/sql/driver"
	"reflect"

	"github.com/lib/pq"
	"github.com/siti-nabila/orm/dialect"
)

func normalizeArgs(d dialect.Dialector, args []any) []any {
	if d == nil || d.Type() != dialect.DialectPostgres || len(args) == 0 {
		return args
	}

	out := make([]any, len(args))
	for i, arg := range args {
		out[i] = normalizePostgresArg(arg)
	}

	return out
}

func normalizePostgresArg(arg any) any {
	if arg == nil {
		return nil
	}

	if _, ok := arg.(driver.Valuer); ok {
		return arg
	}

	v := reflect.ValueOf(arg)
	if !v.IsValid() {
		return arg
	}

	t := v.Type()
	for t.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
		t = v.Type()
	}

	if t.Kind() != reflect.Slice && t.Kind() != reflect.Array {
		return arg
	}

	if t.Elem().Kind() == reflect.Uint8 {
		return arg
	}

	return pq.Array(arg)
}
