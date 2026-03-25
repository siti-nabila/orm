package builder

import "github.com/siti-nabila/orm/internal/mapper"

func GenerateValuesFromMeta(cols []mapper.ColumnMeta) []any {
	out := make([]any, len(cols))

	for idx, v := range cols {
		out[idx] = v.Value
	}

	return out
}
