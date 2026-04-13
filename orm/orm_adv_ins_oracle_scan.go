package orm

import (
	"context"
	"database/sql"

	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
	normalizeerr "github.com/siti-nabila/orm/pkg/normalize_err"
)

func executeOracleAdvInsertScan(
	ctx context.Context,
	o *ORM,
	buildRes builder.InsertAdvancedQueryResult,
) error {
	if len(buildRes.ReturningCols) == 0 {
		return dictionary.ErrAdvInsScanWithoutReturning
	}

	if !buildRes.OracleReturningOut {
		return dictionary.ErrAdvInsOracleReturningBindFailed
	}

	outArgs, assignments, err := prepareOracleReturningBindTargets(
		buildRes.ReturningCols,
		o.Dialect(),
	)
	if err != nil {
		return err
	}

	execArgs := make([]any, 0, len(buildRes.Args)+len(outArgs))
	execArgs = append(execArgs, buildRes.Args...)
	execArgs = append(execArgs, outArgs...)

	if _, err := o.executor.ExecContext(ctx, buildRes.Query, execArgs...); err != nil {
		return normalizeerr.Normalize(o.Dialect().Name(), err)
	}

	return applyScanAssignments(assignments)
}

func prepareOracleReturningBindTargets(
	cols []mapper.ColumnMeta,
	d dialect.Dialector,
) ([]any, []scanAssignment, error) {
	if len(cols) == 0 {
		return nil, nil, dictionary.ErrDBQueryEmpty
	}

	targets := make([]any, 0, len(cols))
	assignments := make([]scanAssignment, 0, len(cols))

	for _, col := range cols {
		field := col.FieldSrc

		if !field.IsValid() || !field.CanAddr() {
			return nil, nil, dictionary.ErrUnaddressableDestError(col.Name)
		}

		target, assignment, err := buildScanTargetForField(d, col.Name, field)
		if err != nil {
			return nil, nil, err
		}

		targets = append(targets, sql.Out{Dest: target})

		if assignment != nil {
			assignments = append(assignments, *assignment)
		}
	}

	return targets, assignments, nil
}
