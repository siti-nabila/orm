package logger

import (
	"fmt"
	"time"

	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
)

type (
	Logger interface {
		Log(query string, d dialect.Dialector, cols []mapper.ColumnMeta, args []any, mode string, duration time.Duration, err error)
		LogDryRun(query string, d dialect.Dialector, cols []mapper.ColumnMeta, args []any, mode string)
	}
	DefaultLogger struct{}
)

func (d DefaultLogger) Log(
	query string,
	dialector dialect.Dialector,
	cols []mapper.ColumnMeta,
	args []any,
	mode string,
	duration time.Duration,
	err error,
) {
	rendered := Interpolate(query, dialector, cols, args...)

	if err != nil {
		fmt.Printf("[ORM][%s][%s] %v | ERROR: %v | %s\n", dialector.Name(), mode, duration, err, rendered)
		return
	}

	fmt.Printf("[ORM][%s][%s] %v | %s\n", dialector.Name(), mode, duration, rendered)
}

func (d DefaultLogger) LogDryRun(
	query string,
	dialector dialect.Dialector,
	cols []mapper.ColumnMeta,
	args []any,
	mode string,
) {
	rendered := Interpolate(query, dialector, cols, args...)
	fmt.Printf("[ORM][DRY_RUN][%s][%s] %s\n", dialector.Name(), mode, rendered)
}
