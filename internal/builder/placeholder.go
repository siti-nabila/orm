package builder

import (
	"strings"

	"github.com/siti-nabila/orm/internal/config"
	"github.com/siti-nabila/orm/internal/dialect"
	"github.com/siti-nabila/orm/internal/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

func GeneratePlaceholderQuery(d dialect.Dialector, mode config.PlaceholderMode, cols []mapper.ColumnMeta) string {
	if len(cols) <= 0 {
		return ""
	}

	out := make([]string, len(cols))
	for idx, v := range cols {
		switch mode {
		case config.PlaceholderByNumber:
			out[idx] = d.PlaceholderByNumber(idx + 1)
		case config.PlaceholderByName:
			out[idx] = d.PlaceholderByName(v.Name)
		default:
			panic(dictionary.ErrDBPlaceholder)
		}
	}

	return strings.Join(out, config.QuerySeperator)
}
