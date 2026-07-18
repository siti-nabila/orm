package query

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
	"github.com/siti-nabila/orm/pkg/helper"
)

type updateQueryExecutor interface {
	ExecUpdateQuery(ctx context.Context, result builder.UpdateQueryResult) (sql.Result, error)
}

func (b *QueryBuilder) buildUpdate() (builder.UpdateQueryResult, error) {
	if b.orm == nil || b.model == nil {
		return builder.UpdateQueryResult{}, dictionary.ErrMustBeStructPtr
	}
	if err := helper.IsAllowedPointerStruct(b.model); err != nil {
		return builder.UpdateQueryResult{}, err
	}

	cfg := b.orm.Config()
	d := b.orm.Dialect()
	mode := b.orm.PlaceholderMode()
	meta, err := mapper.Parse(b.model, cfg.UseSnakeCase)
	if err != nil {
		return builder.UpdateQueryResult{}, err
	}

	setCols := make([]mapper.ColumnMeta, 0, len(meta.Columns))
	for _, col := range meta.Columns {
		if !col.SQLTagged || col.Name == "" || col.PrimaryKey || col.Where {
			continue
		}
		setCols = append(setCols, col)
	}
	if len(setCols) == 0 {
		return builder.UpdateQueryResult{}, dictionary.ErrDBQueryEmpty
	}

	setQuery, err := builder.GenerateUpdateSetQuery(d, mode, cfg.QuoteIdentifier, setCols, 1)
	if err != nil {
		return builder.UpdateQueryResult{}, err
	}

	var (
		whereQuery string
		whereArgs  []any
		whereCols  []mapper.ColumnMeta
		pk         = meta.GetPrimaryKeyColumn()
	)

	if len(b.conditions) > 0 {
		whereQuery, whereArgs, _, err = buildConditions(d, mode, b.conditions, len(setCols)+1)
		if err != nil {
			return builder.UpdateQueryResult{}, err
		}
	} else {
		if pk != nil {
			if helper.IsZero(pk.Value) {
				return builder.UpdateQueryResult{}, dictionary.ErrPrimaryKeyEmpty
			}
			whereCols = []mapper.ColumnMeta{*pk}
		} else {
			whereCols = meta.GetWhereColumns()
			if len(whereCols) == 0 {
				return builder.UpdateQueryResult{}, dictionary.ErrUpdateWithoutWhere
			}
			if err := builder.ValidateWhereColumns(whereCols); err != nil {
				return builder.UpdateQueryResult{}, err
			}
		}

		whereQuery, err = builder.GenerateWhereColumnsQuery(
			d, mode, cfg.QuoteIdentifier, whereCols, len(setCols)+1,
		)
		if err != nil {
			return builder.UpdateQueryResult{}, err
		}
		whereArgs = builder.GenerateValuesFromMeta(whereCols)
	}

	if whereQuery == "" {
		return builder.UpdateQueryResult{}, dictionary.ErrUpdateWithoutWhere
	}

	table := meta.Table
	if cfg.QuoteIdentifier {
		table = d.QuoteIdentifier(table)
	}
	args := builder.GenerateValuesFromMeta(setCols)
	args = append(args, whereArgs...)
	placeholderCols := append([]mapper.ColumnMeta{}, setCols...)
	placeholderCols = append(placeholderCols, whereCols...)

	return builder.UpdateQueryResult{
		Query:           fmt.Sprintf("UPDATE %s SET %s WHERE %s", table, setQuery, whereQuery),
		Args:            args,
		PKColumn:        pk,
		FilteredCols:    setCols,
		PlaceholderCols: placeholderCols,
	}, nil
}

// Updates updates every SQL-tagged model column except primary-key and tagged-where columns.
// Prefer an update-specific struct so zero values are intentionally included in SET.
func (b *QueryBuilder) Updates() (sql.Result, error) {
	res, err := b.buildUpdate()
	if err != nil {
		return nil, err
	}
	executor, ok := b.orm.(updateQueryExecutor)
	if !ok {
		return nil, dictionary.ErrUpdateExecutorUnsupported
	}
	return executor.ExecUpdateQuery(b.ctx, res)
}

func (b *QueryBuilder) DryRunUpdates() (builder.DryRunResult, error) {
	res, err := b.buildUpdate()
	if err != nil {
		return builder.DryRunResult{}, err
	}
	return builder.DryRunResult{
		Query: res.Query,
		Args:  res.Args,
		Mode:  builder.DryRunModeExec,
	}, nil
}
