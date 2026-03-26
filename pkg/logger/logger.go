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

func (d DefaultLogger) LogUpdateQuery(
	query string,
	dialector dialect.Dialector,
	setCols []mapper.ColumnMeta,
	pkCol mapper.ColumnMeta,
	args []any,
	duration time.Duration,
	err error,
) {
	cols := make([]mapper.ColumnMeta, 0, len(setCols)+1)
	cols = append(cols, setCols...)
	cols = append(cols, pkCol)

	interpolatedQuery := Interpolate(query, dialector, cols, args...)
	ms := float64(duration) / float64(time.Millisecond)

	if err != nil {
		fmt.Printf("[SQL] %s | Duration: %.2fms | Error: %v\n", interpolatedQuery, ms, err)
	} else {
		fmt.Printf("[SQL] %s | Duration: %.2fms\n", interpolatedQuery, ms)
	}
}
