package builder

import (
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
) (InsertQueryResult, error) {
	core, err := buildInsertCore(meta, d, cfg, mode)
	if err != nil {
		return InsertQueryResult{}, err
	}

	pk := meta.GetPrimaryKeyColumn()
	query := core.Query

	if returningPk && d.SupportReturning() && pk != nil {
		pkName := pk.Name
		if cfg.QuoteIdentifier {
			pkName = d.QuoteIdentifier(pkName)
		}
		query += " RETURNING " + pkName
	}

	return InsertQueryResult{
		Query:        query,
		Args:         core.Args,
		PKColumn:     pk,
		FilteredCols: core.FilteredCols,
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
