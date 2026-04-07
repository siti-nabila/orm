package logger

import (
	"fmt"
	"strings"
	"time"

	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
)

// var (
// 	pgPlaceholder         = regexp.MustCompile(`\$(\d+)`)
// 	oracleNumPlaceholder  = regexp.MustCompile(`:(\d+)`)
// 	sqlServerPlaceholder  = regexp.MustCompile(`@p(\d+)`)
// 	oracleNamePlaceholder = regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)
// )

func Interpolate(
	query string,
	d dialect.Dialector,
	cols []mapper.ColumnMeta,
	args ...any,
) string {

	if len(args) == 0 {
		return query
	}

	switch d.Type() {

	case dialect.DialectPostgres:
		return interpolateNumbered(query, "$", args...)

	case dialect.DialectOracle:
		// detect apakah :1 atau :name
		if strings.Contains(query, ":1") {
			return interpolateNumbered(query, ":", args...)
		}
		return interpolateNamed(query, cols, args...)

	case dialect.DialectMySQL:
		return interpolateQuestion(query, args...)

	default:
		return query
	}
}

func interpolateNumbered(query, prefix string, args ...any) string {
	for i, arg := range args {
		ph := fmt.Sprintf("%s%d", prefix, i+1)
		query = strings.Replace(query, ph, formatValue(arg), 1)
	}
	return query
}

func interpolateQuestion(query string, args ...any) string {
	var b strings.Builder
	argIdx := 0

	for i := 0; i < len(query); i++ {
		if query[i] == '?' && argIdx < len(args) {
			b.WriteString(formatValue(args[argIdx]))
			argIdx++
			continue
		}
		b.WriteByte(query[i])
	}

	return b.String()
}

func interpolateNamed(
	query string,
	cols []mapper.ColumnMeta,
	args ...any,
) string {

	argMap := make(map[string]any, len(cols))

	for i, c := range cols {
		if i >= len(args) {
			break
		}
		argMap[c.Name] = args[i]
	}

	for name, val := range argMap {
		ph := ":" + name
		query = strings.ReplaceAll(query, ph, formatValue(val))
	}

	return query
}

func formatValue(v any) string {
	switch val := v.(type) {

	case nil:
		return "NULL"

	case string:
		return "'" + strings.ReplaceAll(val, "'", "''") + "'"

	case []byte:
		return "'" + strings.ReplaceAll(string(val), "'", "''") + "'"

	case bool:
		if val {
			return "TRUE"
		}
		return "FALSE"

	case time.Time:
		return "'" + val.Format(time.RFC3339) + "'"

	// slice support (IN clause)
	case []string:
		return joinQuoted(val)

	case []int:
		return joinNumbers(val)

	case []int64:
		return joinNumbers(val)

	case []uint:
		return joinNumbers(val)

	case []uint64:
		return joinNumbers(val)

	case []any:
		return joinAny(val)

	default:
		return fmt.Sprintf("%v", val)
	}
}

func joinQuoted(strs []string) string {
	out := make([]string, len(strs))
	for i, s := range strs {
		out[i] = "'" + strings.ReplaceAll(s, "'", "''") + "'"
	}
	return "(" + strings.Join(out, ",") + ")"
}

func joinNumbers[T ~int | ~int64 | ~uint | ~uint64](nums []T) string {
	out := make([]string, len(nums))
	for i, n := range nums {
		out[i] = fmt.Sprintf("%v", n)
	}
	return "(" + strings.Join(out, ",") + ")"
}

func joinAny(vals []any) string {
	out := make([]string, len(vals))
	for i, v := range vals {
		out[i] = formatValue(v)
	}
	return "(" + strings.Join(out, ",") + ")"
}
