package builder

import (
	"fmt"
	"strings"

	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

func buildOracleInsertAdvancedQuery(
	meta *mapper.Meta,
	d dialect.Dialector,
	cfg config.Config,
	mode config.PlaceholderMode,
	opts InsertBuildOptions,
) (InsertAdvancedQueryResult, error) {
	if opts.OnConflict != nil {
		return buildOracleMergeAdvancedQuery(meta, d, cfg, mode, opts)
	}
	return buildOraclePlainInsertAdvancedQuery(meta, d, cfg, mode, opts)
}

func buildOraclePlainInsertAdvancedQuery(
	meta *mapper.Meta,
	d dialect.Dialector,
	cfg config.Config,
	mode config.PlaceholderMode,
	opts InsertBuildOptions,
) (InsertAdvancedQueryResult, error) {
	core, err := buildInsertCore(meta, d, cfg, mode)
	if err != nil {
		return InsertAdvancedQueryResult{}, err
	}

	query := core.Query
	if len(opts.ReturningCols) > 0 {
		returningSQL, err := buildOracleReturningIntoClause(
			d,
			cfg,
			mode,
			opts.ReturningCols,
			len(core.Args)+1,
		)
		if err != nil {
			return InsertAdvancedQueryResult{}, err
		}
		query += " " + returningSQL
	}

	return InsertAdvancedQueryResult{
		Query:              query,
		Args:               core.Args,
		FilteredCols:       core.FilteredCols,
		ReturningCols:      opts.ReturningCols,
		OracleReturningOut: len(opts.ReturningCols) > 0,
		Mode: func() DryRunMode {
			if len(opts.ReturningCols) > 0 {
				return DryRunModeQuery
			}
			return DryRunModeExec
		}(),
	}, nil
}

func buildOracleMergeAdvancedQuery(
	meta *mapper.Meta,
	d dialect.Dialector,
	cfg config.Config,
	mode config.PlaceholderMode,
	opts InsertBuildOptions,
) (InsertAdvancedQueryResult, error) {
	conf := opts.OnConflict
	if conf == nil {
		return InsertAdvancedQueryResult{}, dictionary.ErrDBQueryEmpty
	}
	if conf.DoNothing {
		return InsertAdvancedQueryResult{}, dictionary.ErrAdvInsConflictUnsupportedAction("Do Nothing")
	}

	insertCols := filterInsertColumns(meta.Columns)
	if len(insertCols) == 0 {
		return InsertAdvancedQueryResult{}, dictionary.ErrDBQueryEmpty
	}

	usingSQL, usingArgs, err := buildOracleMergeUsingSelectFromDual(
		d,
		cfg,
		mode,
		insertCols,
		1,
	)
	if err != nil {
		return InsertAdvancedQueryResult{}, err
	}

	onSQL := buildOracleMergeOnClause(
		d,
		cfg,
		conf.TargetCols,
	)

	updateSQL, updateArgs, nextIndex, err := buildOracleMergeUpdateClause(

		d,
		cfg,
		mode,
		conf.Assignments,
		len(usingArgs)+1,
	)
	if err != nil {
		return InsertAdvancedQueryResult{}, err
	}

	insertSQL := buildOracleMergeInsertClause(
		d,
		cfg,
		insertCols,
	)

	targetTable := quoteConflictName(meta.Table, d, cfg.QuoteIdentifier)

	var sb strings.Builder
	sb.WriteString("MERGE INTO ")
	sb.WriteString(targetTable)
	sb.WriteString(" t ")
	sb.WriteString(usingSQL)
	sb.WriteString(" ")
	sb.WriteString(onSQL)
	sb.WriteString(" ")
	sb.WriteString(updateSQL)
	sb.WriteString(" ")
	sb.WriteString(insertSQL)

	if len(opts.ReturningCols) > 0 {
		returningSQL, err := buildOracleReturningIntoClause(
			d,
			cfg,
			mode,
			opts.ReturningCols,
			nextIndex,
		)
		if err != nil {
			return InsertAdvancedQueryResult{}, err
		}
		sb.WriteString(" ")
		sb.WriteString(returningSQL)
	}

	args := make([]any, 0, len(usingArgs)+len(updateArgs))
	args = append(args, usingArgs...)
	args = append(args, updateArgs...)

	return InsertAdvancedQueryResult{
		Query:              sb.String(),
		Args:               args,
		FilteredCols:       insertCols,
		ReturningCols:      opts.ReturningCols,
		OracleReturningOut: len(opts.ReturningCols) > 0,
		Mode: func() DryRunMode {
			if len(opts.ReturningCols) > 0 {
				return DryRunModeQuery
			}
			return DryRunModeExec
		}(),
	}, nil
}

func buildOracleMergeUsingSelectFromDual(
	d dialect.Dialector,
	cfg config.Config,
	mode config.PlaceholderMode,
	cols []mapper.ColumnMeta,
	startIndex int,
) (string, []any, error) {
	if len(cols) == 0 {
		return "", nil, dictionary.ErrDBQueryEmpty
	}

	selectParts := make([]string, 0, len(cols))
	args := make([]any, 0, len(cols))

	for i, col := range cols {
		ph, err := GeneratePlaceholder(d, mode, startIndex+i, col)
		if err != nil {
			return "", nil, err
		}

		alias := quoteConflictName(col.Name, d, cfg.QuoteIdentifier)
		selectParts = append(selectParts, fmt.Sprintf("%s AS %s", ph, alias))
		args = append(args, col.Value)
	}

	sql := fmt.Sprintf(
		"USING (SELECT %s FROM dual) s",
		strings.Join(selectParts, ", "),
	)

	return sql, args, nil
}

func buildOracleMergeOnClause(
	d dialect.Dialector,
	cfg config.Config,
	targetCols []mapper.ColumnMeta,
) string {
	parts := make([]string, 0, len(targetCols))

	for _, col := range targetCols {
		colName := quoteConflictName(col.Name, d, cfg.QuoteIdentifier)
		parts = append(parts, fmt.Sprintf("t.%s = s.%s", colName, colName))
	}

	return fmt.Sprintf("ON (%s)", strings.Join(parts, " AND "))
}

func buildOracleMergeUpdateClause(
	d dialect.Dialector,
	cfg config.Config,
	mode config.PlaceholderMode,
	assignments []ResolvedConflictAssignment,
	startIndex int,
) (string, []any, int, error) {
	if len(assignments) == 0 {
		return "", nil, startIndex, dictionary.ErrDBQueryEmpty
	}

	updateParts := make([]string, 0, len(assignments))
	args := make([]any, 0, len(assignments))
	nextIndex := startIndex

	for _, a := range assignments {
		sqlPart, sqlArgs, next, err := buildOracleMergeAssignment(
			d,
			cfg,
			mode,
			a,
			nextIndex,
		)
		if err != nil {
			return "", nil, nextIndex, err
		}

		updateParts = append(updateParts, sqlPart)
		args = append(args, sqlArgs...)
		nextIndex = next
	}

	return "WHEN MATCHED THEN UPDATE SET " + strings.Join(updateParts, ", "), args, nextIndex, nil
}

func buildOracleMergeAssignment(
	d dialect.Dialector,
	cfg config.Config,
	mode config.PlaceholderMode,
	a ResolvedConflictAssignment,
	nextIndex int,
) (string, []any, int, error) {
	colName := quoteConflictName(a.ColumnMeta.Name, d, cfg.QuoteIdentifier)

	switch a.Mode {
	case ConflictAssignInserted:
		return fmt.Sprintf("t.%s = s.%s", colName, colName), nil, nextIndex, nil

	case ConflictAssignValue:
		ph, err := GeneratePlaceholder(d, mode, nextIndex, a.ColumnMeta)
		if err != nil {
			return "", nil, nextIndex, err
		}
		return fmt.Sprintf("t.%s = %s", colName, ph), []any{a.Value}, nextIndex + 1, nil

	case ConflictAssignInc:
		if a.RefColumn == nil {
			return "", nil, nextIndex, dictionary.ErrAdvInsIncMissingRefColumn
		}

		refName := quoteConflictName(a.RefColumn.Name, d, cfg.QuoteIdentifier)
		ph, err := GeneratePlaceholder(d, mode, nextIndex, a.ColumnMeta)
		if err != nil {
			return "", nil, nextIndex, err
		}

		return fmt.Sprintf("t.%s = t.%s + %s", colName, refName, ph), []any{a.Value}, nextIndex + 1, nil

	default:
		return "", nil, nextIndex, dictionary.ErrAdvInsInvalidMode
	}
}

func buildOracleMergeInsertClause(
	d dialect.Dialector,
	cfg config.Config,
	insertCols []mapper.ColumnMeta,
) string {
	colNames := make([]string, 0, len(insertCols))
	sourceCols := make([]string, 0, len(insertCols))

	for _, col := range insertCols {
		colName := quoteConflictName(col.Name, d, cfg.QuoteIdentifier)
		colNames = append(colNames, colName)
		sourceCols = append(sourceCols, "s."+colName)
	}

	return fmt.Sprintf(
		"WHEN NOT MATCHED THEN INSERT (%s) VALUES (%s)",
		strings.Join(colNames, ", "),
		strings.Join(sourceCols, ", "),
	)
}

func buildOracleReturningIntoClause(
	d dialect.Dialector,
	cfg config.Config,
	mode config.PlaceholderMode,
	returningCols []mapper.ColumnMeta,
	startIndex int,
) (string, error) {
	if len(returningCols) == 0 {
		return "", nil
	}

	colList := GenerateColumnListQuery(
		d,
		cfg.QuoteIdentifier,
		returningCols,
	)

	placeholders := make([]string, 0, len(returningCols))
	for i, col := range returningCols {
		ph, err := GeneratePlaceholder(d, mode, startIndex+i, col)
		if err != nil {
			return "", err
		}
		placeholders = append(placeholders, ph)
	}

	return fmt.Sprintf(
		"RETURNING %s INTO %s",
		colList,
		strings.Join(placeholders, ", "),
	), nil
}
