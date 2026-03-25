package logger

import (
	"fmt"
	"time"

	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
)

type (
	Logger interface {
		Log(query string, d dialect.Dialector, cols []mapper.ColumnMeta, args []any, duration time.Duration, err error)
	}
	DefaultLogger struct{}
)

func (d DefaultLogger) Log(query string, dialector dialect.Dialector, cols []mapper.ColumnMeta, args []any, duration time.Duration, err error) {
	interpolatedQuery := Interpolate(query, dialector, cols, args...)
	ms := float64(duration.Microseconds()) / 1000.0
	if err != nil {
		fmt.Printf("[SQL] %s | Duration: %.2fms | Error: %v\n", interpolatedQuery, ms, err)
	} else {
		fmt.Printf("[SQL] %s | Duration: %.2fms\n", interpolatedQuery, ms)
	}
}
