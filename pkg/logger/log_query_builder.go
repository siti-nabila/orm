package logger

import (
	"database/sql/driver"
	"fmt"
	"reflect"
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

const (
	maxLogArgLength  = 120
	truncatedLogMark = "...(truncated)..."
)

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
		return interpolateNumbered(query, "$", d.Type(), args...)

	case dialect.DialectOracle:
		// detect apakah :1 atau :name
		if strings.Contains(query, ":1") {
			return interpolateNumbered(query, ":", d.Type(), args...)
		}
		return interpolateNamed(query, d.Type(), cols, args...)

	case dialect.DialectMySQL:
		return interpolateQuestion(query, d.Type(), args...)

	default:
		return query
	}
}

func interpolateNumbered(query, prefix string, dType dialect.DialectType, args ...any) string {
	for i, arg := range args {
		ph := fmt.Sprintf("%s%d", prefix, i+1)
		query = strings.Replace(query, ph, formatValueForDialect(arg, dType), 1)
	}
	return query
}

func interpolateQuestion(query string, dType dialect.DialectType, args ...any) string {
	var b strings.Builder
	argIdx := 0

	for i := 0; i < len(query); i++ {
		if query[i] == '?' && argIdx < len(args) {
			b.WriteString(formatValueForDialect(args[argIdx], dType))
			argIdx++
			continue
		}
		b.WriteByte(query[i])
	}

	return b.String()
}

func interpolateNamed(
	query string,
	dType dialect.DialectType,
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
		query = strings.ReplaceAll(query, ph, formatValueForDialect(val, dType))
	}

	return query
}

func formatValue(v any) string {
	return formatValueForDialect(v, "")
}

func formatValueForDialect(v any, dType dialect.DialectType) string {
	if dType == dialect.DialectPostgres {
		if rendered, ok := formatPostgresArray(v); ok {
			return rendered
		}
	}

	v = normalizeLogValue(v)
	switch val := v.(type) {

	case nil:
		return "NULL"

	case string:
		return truncateLogValue("'" + strings.ReplaceAll(val, "'", "''") + "'")

	case []byte:
		return truncateLogValue("'" + strings.ReplaceAll(string(val), "'", "''") + "'")

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

	case []int8:
		return joinNumbers(val)

	case []int16:
		return joinNumbers(val)

	case []int32:
		return joinNumbers(val)

	case []int:
		return joinNumbers(val)

	case []int64:
		return joinNumbers(val)

	case []uint16:
		return joinNumbers(val)

	case []uint32:
		return joinNumbers(val)

	case []uint:
		return joinNumbers(val)

	case []uint64:
		return joinNumbers(val)

	case []any:
		return joinAny(val)

	default:
		return truncateLogValue(fmt.Sprintf("%v", val))
	}
}

func normalizeLogValue(v any) any {
	if v == nil {
		return nil
	}

	if val, ok := driverValue(v); ok {
		return val
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil
	}

	for rv.Kind() == reflect.Interface || rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}

		rv = rv.Elem()
		if rv.CanInterface() {
			if val, ok := driverValue(rv.Interface()); ok {
				return val
			}
		}
	}

	if !rv.CanInterface() {
		return v
	}

	return rv.Interface()
}

func driverValue(v any) (any, bool) {
	valuer, ok := v.(driver.Valuer)
	if !ok {
		return nil, false
	}

	val, err := valuer.Value()
	if err != nil {
		return nil, false
	}

	return val, true
}

func truncateLogValue(s string) string {
	return truncateLogValueTo(s, maxLogArgLength)
}

func truncateLogValueTo(s string, limit int) string {
	if limit <= 0 {
		return ""
	}

	if len(s) <= limit {
		return s
	}

	if limit <= len(truncatedLogMark) {
		return s[:limit]
	}

	remaining := limit - len(truncatedLogMark)
	headLen := remaining / 2
	tailLen := remaining - headLen

	return s[:headLen] + truncatedLogMark + s[len(s)-tailLen:]
}

func formatPostgresArray(v any) (string, bool) {
	if v == nil {
		return "", false
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return "", false
	}

	rt := rv.Type()
	for rt.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return "NULL", true
		}
		rv = rv.Elem()
		rt = rv.Type()
	}

	if rt.Kind() != reflect.Slice && rt.Kind() != reflect.Array {
		return "", false
	}

	if rt.Elem().Kind() == reflect.Uint8 {
		return "", false
	}

	return joinValues("ARRAY[", "]", rv.Len(), func(i int) string {
		return formatValueForDialect(rv.Index(i).Interface(), dialect.DialectPostgres)
	}), true
}

func joinQuoted(strs []string) string {
	return joinValues("(", ")", len(strs), func(i int) string {
		return "'" + strings.ReplaceAll(strs[i], "'", "''") + "'"
	})
}

func joinNumbers[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint16 | ~uint32 | ~uint64](nums []T) string {
	return joinValues("(", ")", len(nums), func(i int) string {
		return fmt.Sprintf("%v", nums[i])
	})
}

func joinAny(vals []any) string {
	return joinValues("(", ")", len(vals), func(i int) string {
		return formatValue(vals[i])
	})
}

func joinValues(prefix, suffix string, length int, format func(int) string) string {
	var b strings.Builder
	b.WriteString(prefix)

	for i := 0; i < length; i++ {
		if i > 0 {
			b.WriteByte(',')
		}

		b.WriteString(format(i))
		if b.Len()+len(suffix) > maxLogArgLength {
			return joinTruncatedValues(prefix, suffix, length, format)
		}
	}

	b.WriteString(suffix)
	return b.String()
}

func joinTruncatedValues(prefix, suffix string, length int, format func(int) string) string {
	if length == 0 {
		return prefix + suffix
	}

	if length == 1 {
		return truncateLogValue(prefix + format(0) + suffix)
	}

	middle := "," + truncatedLogMark + ","
	budget := maxLogArgLength - len(prefix) - len(suffix) - len(middle)
	if budget <= 0 {
		return truncateLogValue(prefix + truncatedLogMark + suffix)
	}

	frontBudget := budget / 2
	backBudget := budget - frontBudget

	frontParts := make([]string, 0)
	frontLen := 0
	for i := 0; i < length; i++ {
		part := format(i)
		addLen := len(part)
		if len(frontParts) > 0 {
			addLen++
		}

		if frontLen+addLen > frontBudget {
			if len(frontParts) == 0 && frontBudget > 0 {
				frontParts = append(frontParts, truncateLogValueTo(part, frontBudget))
			}
			break
		}

		frontParts = append(frontParts, part)
		frontLen += addLen
	}

	backParts := make([]string, 0)
	backLen := 0
	for i := length - 1; i >= len(frontParts); i-- {
		part := format(i)
		addLen := len(part)
		if len(backParts) > 0 {
			addLen++
		}

		if backLen+addLen > backBudget {
			if len(backParts) == 0 && backBudget > 0 {
				backParts = append(backParts, truncateLogValueTo(part, backBudget))
			}
			break
		}

		backParts = append(backParts, part)
		backLen += addLen
	}

	for i, j := 0, len(backParts)-1; i < j; i, j = i+1, j-1 {
		backParts[i], backParts[j] = backParts[j], backParts[i]
	}

	out := prefix + strings.Join(frontParts, ",") + middle + strings.Join(backParts, ",") + suffix
	return truncateLogValue(out)
}
