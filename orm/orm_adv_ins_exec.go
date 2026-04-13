package orm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"time"

	"github.com/godev90/validator/faults"
	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

func (c *CreateCommand) WithReturning(columns ...string) *CreateCommand {
	c.opts.Returning = append(c.opts.Returning, columns...)
	return c
}

func (c *CreateCommand) WithOnConflict(conflict OnConflict) *CreateCommand {
	x := conflict
	c.opts.OnConflict = &x
	return c
}

func (c *CreateCommand) Exec() error {
	return c.exec(false)
}

func (c *CreateCommand) Scan() error {
	return c.exec(true)
}

func (c *CreateCommand) ScanInto(dest ...any) error {
	if len(dest) == 0 {
		return dictionary.ErrDBScanIntoEmptyDest
	}

	c.scanIntoDest = append([]any(nil), dest...)
	return c.exec(true)
}
func (c *CreateCommand) exec(expectReturning bool) error {
	var (
		err   error
		start = time.Now()
	)

	if c == nil || c.orm == nil {
		return dictionary.ErrDBQueryEmpty
	}

	if expectReturning {
		if len(c.opts.Returning) == 0 {
			return dictionary.ErrAdvInsScanWithoutReturning
		}
	} else {
		if len(c.opts.Returning) > 0 {
			return dictionary.ErrAdvInsExecWithReturning
		}
	}

	if expectReturning && len(c.scanIntoDest) > 0 {
		if err := validateScanInto(c.scanIntoDest); err != nil {
			return err
		}
		if len(c.scanIntoDest) != len(c.opts.Returning) {
			return dictionary.ErrScanIntoColCountMismatch(len(c.opts.Returning), len(c.scanIntoDest))
		}
	}

	meta, err := mapper.Parse(c.v, c.orm.config.UseSnakeCase)
	if err != nil {
		return err
	}

	d := c.orm.Dialect()

	resolved, err := resolveCreateBuildOptions(meta, c.opts, d)
	if err != nil {
		return err
	}

	buildRes, err := builder.BuildInsertQueryWithOptions(
		meta,
		d,
		c.orm.config,
		c.orm.placeholderMode(),
		resolved.BuildOpts,
	)
	if err != nil {
		return err
	}

	defer func() {
		if c.orm.logger != nil {
			c.orm.logger.Log(
				buildRes.Query,
				d,
				buildRes.FilteredCols,
				buildRes.Args,
				buildRes.Mode.String(),
				time.Since(start),
				err,
			)
		}
	}()

	switch d.Type() {
	case dialect.DialectPostgres:
		err = execAdvInsPostgres(
			c.ctx,
			c.orm,
			buildRes,
			expectReturning,
			c.scanIntoDest,
		)
	case dialect.DialectMySQL:
		err = execAdvInsMySQL(
			c.ctx,
			c.orm,
			meta,
			buildRes,
			resolved,
			expectReturning,
			c.scanIntoDest,
		)
	case dialect.DialectOracle:
		err = execAdvInsOracle(
			c.ctx,
			c.orm,
			buildRes,
			expectReturning,
			c.scanIntoDest,
		)
	default:
		err = dictionary.ErrUnsupportedDialect
	}

	return err
}

// -----------------
func execAdvInsPostgres(
	ctx context.Context,
	orm *ORM,
	buildRes builder.InsertAdvancedQueryResult,
	expectReturning bool,
	dest []any,
) error {
	if !expectReturning {
		_, err := orm.executor.ExecContext(ctx, buildRes.Query, buildRes.Args...)
		return err
	}

	row := orm.executor.QueryRowContext(ctx, buildRes.Query, buildRes.Args...)

	if len(dest) > 0 {
		return row.Scan(dest...)
	}

	targets, err := buildReturningScanTargets(buildRes.ReturningCols)
	if err != nil {
		return err
	}

	return row.Scan(targets...)
}

func execAdvInsMySQL(
	ctx context.Context,
	orm *ORM,
	meta *mapper.Meta,
	buildRes builder.InsertAdvancedQueryResult,
	resolved createBuildResolved,
	expectReturning bool,
	dest []any,
) error {
	if !expectReturning {
		_, err := orm.executor.ExecContext(ctx, buildRes.Query, buildRes.Args...)
		return err
	}

	_, err := orm.executor.ExecContext(ctx, buildRes.Query, buildRes.Args...)
	if err != nil {
		return err
	}

	selectRes, err := buildMySQLReturningSelectQuery(
		meta,
		buildRes,
		resolved,
		orm,
	)
	if err != nil {
		return err
	}

	row := orm.executor.QueryRowContext(ctx, selectRes.Query, selectRes.Args...)

	if len(dest) > 0 {
		return row.Scan(dest...)
	}

	targets, err := buildReturningScanTargets(selectRes.TargetCols)
	if err != nil {
		return err
	}

	return row.Scan(targets...)
}

func execAdvInsOracle(
	ctx context.Context,
	orm *ORM,
	buildRes builder.InsertAdvancedQueryResult,
	expectReturning bool,
	dest []any,
) error {
	if !expectReturning {
		_, err := orm.executor.ExecContext(ctx, buildRes.Query, buildRes.Args...)
		return err
	}

	var outTargets []any
	if len(dest) > 0 {
		outTargets = dest
	} else {
		var err error
		outTargets, err = buildReturningScanTargets(buildRes.ReturningCols)
		if err != nil {
			return err
		}
	}

	args := make([]any, 0, len(buildRes.Args)+len(outTargets))
	args = append(args, buildRes.Args...)
	for _, t := range outTargets {
		args = append(args, sql.Out{Dest: t})
	}

	_, err := orm.executor.ExecContext(ctx, buildRes.Query, args...)
	return err
}

func validateScanInto(dest []any) error {
	errs := faults.Errors{}

	for i, d := range dest {
		if d == nil {
			errs[rowKey(i)+":nil"] = dictionary.ErrDBScanNilDest
			continue
		}

		rv := reflect.ValueOf(d)
		if rv.Kind() != reflect.Ptr || rv.IsNil() {
			errs[rowKey(i)+":not_pointer"] = dictionary.ErrDBScanNotPointerDest
		}
	}

	if len(errs) != 0 {
		return errs
	}
	return nil
}

func rowKey(i int) string {
	return fmt.Sprintf("row %d", i)
}

func buildMySQLReturningSelectQuery(
	meta *mapper.Meta,
	buildRes builder.InsertAdvancedQueryResult,
	resolved createBuildResolved,
	orm *ORM,
) (builder.ReturningSelectQueryResult, error) {
	if len(buildRes.ReturningCols) == 0 {
		return builder.ReturningSelectQueryResult{}, dictionary.ErrDBQueryEmpty
	}
	if len(resolved.TargetCols) == 0 {
		return builder.ReturningSelectQueryResult{}, dictionary.ErrAdvInsReturningNotFound
	}

	return builder.BuildReturningSelectQuery(
		meta,
		orm.Dialect(),
		buildRes.ReturningCols,
		resolved.TargetCols,
		orm.config,
		orm.placeholderMode(),
	)
}
