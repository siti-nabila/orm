package builder

import (
	"fmt"

	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/helper"
)

func BuildInsertQuery(
	meta *mapper.Meta,
	d dialect.Dialector,
	cfg config.Config,
	mode config.PlaceholderMode,
	returningPk bool,
) (string, []any, *mapper.ColumnMeta, []mapper.ColumnMeta) {

	cols := filterInsertColumnsQuery(meta.Columns)
	pk := meta.GetPrimaryKeyColumn()

	if len(cols) == 0 {
		return "", nil, pk, cols
	}

	colList := GenerateColumnListQuery(
		d,
		cfg.QuoteIdentifier,
		cols,
	)

	placeholders := GeneratePlaceholderQuery(
		d,
		mode,
		cols,
	)

	args := GenerateValuesFromMeta(cols)

	table := meta.Table
	if cfg.QuoteIdentifier {
		table = d.QuoteIdentifier(table)
	}

	query := fmt.Sprintf(
		`INSERT INTO %s (%s) VALUES (%s)`,
		table,
		colList,
		placeholders,
	)

	if returningPk && d.SupportReturning() && pk != nil {
		pkName := pk.Name
		if cfg.QuoteIdentifier {
			pkName = d.QuoteIdentifier(pkName)
		}
		query += " RETURNING " + pkName
	}

	return query, args, pk, cols
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
