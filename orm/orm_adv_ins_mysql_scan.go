package orm

import (
	"context"

	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

func executeMySQLAdvInsertScan(
	ctx context.Context,
	o *ORM,
	meta *mapper.Meta,
	buildRes builder.InsertAdvancedQueryResult,
	resolved createBuildResolved,
) error {

	if len(resolved.ReturningCols) == 0 {
		return dictionary.ErrAdvInsScanWithoutReturning
	}

	if len(resolved.TargetCols) == 0 {
		return dictionary.ErrAdvInsTargetColumnEmpty
	}

	// step 1: insert
	if _, err := o.executor.ExecContext(ctx, buildRes.Query, buildRes.Args...); err != nil {
		return err
	}

	// step 2: select
	selectRes, err := builder.BuildReturningSelectQuery(
		meta,
		o.Dialect(),
		resolved.ReturningCols,
		resolved.TargetCols,
		o.config,
		o.placeholderMode(),
	)
	if err != nil {
		return err
	}

	// step 3: scan
	targets, assigns, err := prepareAdvInsScanTargets(selectRes.ReturningCols, o.Dialect())
	if err != nil {
		return err
	}

	if err := o.executor.QueryRowContext(ctx, selectRes.Query, selectRes.Args...).Scan(targets...); err != nil {
		return err
	}

	return applyScanAssignments(assigns)
}
