package builder

import (
	"fmt"
	"strings"

	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

func BuildReturningSelectQuery(
	meta *mapper.Meta,
	d dialect.Dialector,
	returningCols []mapper.ColumnMeta,
	targetCols []mapper.ColumnMeta,
	cfg config.Config,
	mode config.PlaceholderMode,
) (ReturningSelectQueryResult, error) {
	if meta == nil {
		return ReturningSelectQueryResult{}, dictionary.ErrDBQueryEmpty
	}
	if len(returningCols) == 0 {
		return ReturningSelectQueryResult{}, dictionary.ErrDBQueryEmpty
	}
	if len(targetCols) == 0 {
		return ReturningSelectQueryResult{}, dictionary.ErrDBQueryEmpty
	}

	selectCols := GenerateColumnListQuery(
		d,
		cfg.QuoteIdentifier,
		returningCols,
	)

	tableName := meta.Table
	if cfg.QuoteIdentifier {
		tableName = d.QuoteIdentifier(tableName)
	}

	whereSQL, args, err := buildReturningWhereClause(
		d,
		cfg,
		mode,
		targetCols,
		1,
	)
	if err != nil {
		return ReturningSelectQueryResult{}, err
	}

	query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s LIMIT 1",
		selectCols,
		tableName,
		whereSQL,
	)

	return ReturningSelectQueryResult{
		Query:         query,
		Args:          args,
		ReturningCols: returningCols,
		TargetCols:    targetCols,
	}, nil
}

func buildReturningWhereClause(
	d dialect.Dialector,
	cfg config.Config,
	mode config.PlaceholderMode,
	targetCols []mapper.ColumnMeta,
	startIndex int,
) (string, []any, error) {
	if len(targetCols) == 0 {
		return "", nil, dictionary.ErrDBQueryEmpty
	}

	parts := make([]string, 0, len(targetCols))
	args := make([]any, 0, len(targetCols))

	for i, col := range targetCols {
		colName := col.Name
		if cfg.QuoteIdentifier {
			colName = d.QuoteIdentifier(colName)
		}

		ph, err := GeneratePlaceholder(d, mode, startIndex+i, col)
		if err != nil {
			return "", nil, err
		}

		parts = append(parts, fmt.Sprintf("%s = %s", colName, ph))
		args = append(args, col.Value)
	}

	return strings.Join(parts, " AND "), args, nil
}
