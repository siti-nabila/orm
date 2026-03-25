package logger

import (
	"fmt"

	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
)

type (
	Logger interface {
		Log(query string, d dialect.Dialector, cols []mapper.ColumnMeta, args []any)
	}
	DefaultLogger struct{}
)

func (d DefaultLogger) Log(query string, dialector dialect.Dialector, cols []mapper.ColumnMeta, args []any) {
	interpolatedQuery := Interpolate(query, dialector, cols, args...)
	fmt.Printf("[SQL] %s\n", interpolatedQuery)
}
