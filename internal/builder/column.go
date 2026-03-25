package builder

import (
	"strings"

	"github.com/siti-nabila/orm/internal/config"
	"github.com/siti-nabila/orm/internal/dialect"
	"github.com/siti-nabila/orm/internal/mapper"
)

func GenerateColumnListQuery(d dialect.Dialector, quote bool, cols []mapper.ColumnMeta) string {
	if len(cols) <= 0 {
		return ""
	}
	out := make([]string, len(cols))
	for idx, v := range cols {
		name := v.Name
		if quote {
			name = d.QuoteIdentifier(name)
		}
		out[idx] = name

	}
	return strings.Join(out, config.QuerySeperator)
}
