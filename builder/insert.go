package builder

import (
	"fmt"

	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
	"github.com/siti-nabila/orm/pkg/helper"
)

func BuildInsertQuery(
	meta *mapper.Meta,
	d dialect.Dialector,
	cfg config.Config,
	mode config.PlaceholderMode,
	returningPk bool,
) (InsertQueryResult, error) {

	cols := filterInsertColumns(meta.Columns)
	pk := meta.GetPrimaryKeyColumn()

	if len(cols) == 0 {
		return InsertQueryResult{}, dictionary.ErrDBQueryEmpty
	}

	colList := GenerateColumnListQuery(
		d,
		cfg.QuoteIdentifier,
		cols,
	)

	placeholders, err := GeneratePlaceholderQuery(
		d,
		mode,
		cols,
	)
	if err != nil {
		return InsertQueryResult{}, err
	}

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

	return InsertQueryResult{
		Query:        query,
		Args:         args,
		PKColumn:     pk,
		FilteredCols: cols,
	}, nil
}

// skip kolom primary key jika nilainya zero value
func filterInsertColumns(
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
