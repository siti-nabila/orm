package builder

import (
	"fmt"

	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

type (
	insertCoreResult struct {
		Query        string
		Args         []any
		FilteredCols []mapper.ColumnMeta
	}
)

func buildInsertCore(meta *mapper.Meta,
	d dialect.Dialector,
	cfg config.Config,
	mode config.PlaceholderMode,
) (insertCoreResult, error) {
	cols := filterInsertColumns(meta.Columns)

	if len(cols) == 0 {
		return insertCoreResult{}, dictionary.ErrDBQueryEmpty
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
		return insertCoreResult{}, err
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

	return insertCoreResult{
		Query:        query,
		Args:         args,
		FilteredCols: cols,
	}, nil
}
