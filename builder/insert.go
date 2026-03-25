package builder

import (
	"fmt"

	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/helper"
)

func BuildInsertQuery(meta *mapper.Meta, d dialect.Dialector, cfg config.Config, mode config.PlaceholderMode, returningPk bool) (string, []any, *mapper.ColumnMeta) {
	// initialize colums, values, and table
	cols := filterInsertColumnsQuery(meta.Columns)
	colList := GenerateColumnListQuery(d, cfg.QuoteIdentifier, cols)
	values := GenerateValuesFromMeta(cols)
	table := meta.Table
	pk := meta.GetPrimaryKeyColumn()

	// adding quote to table name if needed
	if cfg.QuoteIdentifier {
		table = d.QuoteIdentifier(table)
	}

	query := fmt.Sprintf(`INSERT INTO %s(%s) VALUES (%s)`, table, colList, values)

	if returningPk && d.SupportReturning() && pk != nil {
		pkName := pk.Name
		if cfg.QuoteIdentifier {
			pkName = d.QuoteIdentifier(pk.Name)
		}
		query += " RETURNING " + pkName
	}

	return query, values, pk
}

// fungsi untuk skip kolom primary key dan bernilai nil atau kosong
func filterInsertColumnsQuery(
	cols []mapper.ColumnMeta,
) []mapper.ColumnMeta {

	out := make([]mapper.ColumnMeta, 0, len(cols))

	for _, c := range cols {

		if c.PrimaryKey && helper.IsZero(c.Value) {
			continue
		}

		out = append(out, c)
	}

	return out
}
