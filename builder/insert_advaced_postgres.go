package builder

import (
	"fmt"
	"strings"

	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

func buildPostgresInsertAdvancedQuery(
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
	args := append([]any{}, core.Args...)

	if opts.OnConflict != nil {
		conflictSQL, conflictArgs, err := buildPostgresConflictClause(
			meta,
			d,
			cfg,
			mode,
			opts.OnConflict,
			len(args)+1,
		)
		if err != nil {
			return InsertAdvancedQueryResult{}, err
		}

		if conflictSQL != "" {
			query += " " + conflictSQL
			args = append(args, conflictArgs...)
		}
	}

	if len(opts.ReturningCols) > 0 {
		query += " RETURNING " + GenerateColumnListQuery(
			d,
			cfg.QuoteIdentifier,
			opts.ReturningCols,
		)
	}

	return InsertAdvancedQueryResult{
		Query:         query,
		Args:          args,
		FilteredCols:  core.FilteredCols,
		ReturningCols: opts.ReturningCols,
		Mode: func() DryRunMode {
			if len(opts.ReturningCols) > 0 {
				return DryRunModeQueryRow
			}
			return DryRunModeExec
		}(),
	}, nil
}

func buildPostgresConflictClause(
	meta *mapper.Meta,
	d dialect.Dialector,
	cfg config.Config,
	mode config.PlaceholderMode,
	conf *OnConflictClause,
	startIndex int,
) (string, []any, error) {
	if conf == nil || len(conf.TargetCols) == 0 {
		return "", nil, nil
	}

	targetCols := GenerateColumnListQuery(
		d,
		cfg.QuoteIdentifier,
		conf.TargetCols,
	)

	if conf.DoNothing {
		return fmt.Sprintf("ON CONFLICT (%s) DO NOTHING", targetCols), nil, nil
	}

	tableName := quoteConflictName(meta.Table, d, cfg.QuoteIdentifier)
	assignmentsSQL := make([]string, 0, len(conf.Assignments))
	args := make([]any, 0, len(conf.Assignments))
	nextIndex := startIndex

	for _, a := range conf.Assignments {
		sqlPart, sqlArgs, next, err := buildPostgresConflictAssignment(
			d,
			cfg,
			mode,
			tableName,
			a,
			nextIndex,
		)
		if err != nil {
			return "", nil, err
		}
		if sqlPart == "" {
			return "", nil, nil
		}

		assignmentsSQL = append(assignmentsSQL, sqlPart)
		args = append(args, sqlArgs...)
		nextIndex = next
	}

	if len(assignmentsSQL) == 0 {
		return "", nil, nil
	}

	sql := fmt.Sprintf(
		"ON CONFLICT (%s) DO UPDATE SET %s",
		targetCols,
		strings.Join(assignmentsSQL, ", "),
	)

	return sql, args, nil
}

func buildPostgresConflictAssignment(
	d dialect.Dialector,
	cfg config.Config,
	mode config.PlaceholderMode,
	tableName string,
	a ResolvedConflictAssignment,
	nextIndex int,
) (string, []any, int, error) {
	colName := quoteConflictName(a.ColumnMeta.Name, d, cfg.QuoteIdentifier)

	switch a.Mode {
	case ConflictAssignInserted:
		return fmt.Sprintf("%s = EXCLUDED.%s", colName, colName), nil, nextIndex, nil

	case ConflictAssignValue:
		ph, err := GeneratePlaceholder(d, mode, nextIndex, a.ColumnMeta)
		if err != nil {
			return "", nil, nextIndex, err
		}

		return fmt.Sprintf("%s = %s", colName, ph), []any{a.Value}, nextIndex + 1, nil

	case ConflictAssignInc:
		if a.RefColumn == nil {
			return "", nil, nextIndex, dictionary.ErrAdvInsIncMissingRefColumn
		}

		refName := quoteConflictName(a.RefColumn.Name, d, cfg.QuoteIdentifier)
		ph, err := GeneratePlaceholder(d, mode, nextIndex, a.ColumnMeta)
		if err != nil {
			return "", nil, nextIndex, err
		}

		return fmt.Sprintf(
			"%s = %s.%s + %s",
			colName,
			tableName,
			refName,
			ph,
		), []any{a.Value}, nextIndex + 1, nil

	default:
		return "", nil, nextIndex, dictionary.ErrAdvInsInvalidMode
	}
}
