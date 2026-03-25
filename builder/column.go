package builder

import (
	"strings"

	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
)

func RenderColumnNames(
	d dialect.Dialector,
	quote bool,
	cols []mapper.ColumnMeta,
) []string {
	if len(cols) == 0 {
		return nil
	}

	out := make([]string, len(cols))
	for i, c := range cols {
		name := c.Name
		if quote {
			name = d.QuoteIdentifier(name)
		}
		out[i] = name
	}

	return out
}

func GenerateColumnListQuery(
	d dialect.Dialector,
	quote bool,
	cols []mapper.ColumnMeta,
) string {
	names := RenderColumnNames(d, quote, cols)
	return strings.Join(names, config.QuerySeperator)
}
