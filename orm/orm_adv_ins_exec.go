package orm

import (
	"context"
	"fmt"
	"reflect"
	"time"

	// "github.com/godev90/validator/faults"
	errorpackage "github.com/siti-nabila/error-package"
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

func (c *CreateCommand) Scan(dest ...any) error {
	if len(dest) > 1 {
		return dictionary.ErrDBScanUnsupportedDest
	}
	if len(dest) == 1 {
		if dest[0] == nil {
			return dictionary.ErrDBScanNilDest
		}

		rv := reflect.ValueOf(dest[0])
		if rv.Kind() != reflect.Ptr || rv.IsNil() {
			return dictionary.ErrDBScanNotPointerDest
		}
		if rv.Elem().Kind() != reflect.Struct {
			return dictionary.ErrMustBeStructPtr
		}

		c.scanDest = dest[0]
	}
	return c.exec(true)
}

func (c *CreateCommand) ScanInto(dest ...any) error {
	if len(dest) == 0 {
		return dictionary.ErrDBScanIntoEmptyDest
	}

	c.scanIntoDest = append([]any(nil), dest...)
	return c.exec(true)
}

func (c *CreateCommand) validateReturningMode(expectReturning bool) error {
	if expectReturning {
		if len(c.opts.Returning) == 0 {
			return dictionary.ErrAdvInsScanWithoutReturning
		}
	} else {
		if len(c.opts.Returning) > 0 {
			return dictionary.ErrAdvInsExecWithReturning
		}
	}
	return nil
}

func (c *CreateCommand) resolveScanDest() error {
	if c.scanDest != nil {
		targets, err := resolveStructScanTargets(c.scanDest, c.opts.Returning, c.orm.config.UseSnakeCase)
		if err != nil {
			return err
		}
		c.scanIntoDest = targets
		return nil
	}

	if len(c.scanIntoDest) > 0 {
		if err := validateScanInto(c.scanIntoDest); err != nil {
			return err
		}
		if len(c.scanIntoDest) != len(c.opts.Returning) {
			return dictionary.ErrScanIntoColCountMismatch(len(c.opts.Returning), len(c.scanIntoDest))
		}
	}

	return nil
}

func (c *CreateCommand) prepareCreateWith() (
	builder.InsertAdvancedQueryResult,
	createBuildResolved,
	*mapper.Meta,
	dialect.Dialector,
	error,
) {
	meta, err := mapper.Parse(c.v, c.orm.config.UseSnakeCase)
	if err != nil {
		return builder.InsertAdvancedQueryResult{}, createBuildResolved{}, nil, nil, err
	}

	d := c.orm.Dialect()

	resolved, err := resolveCreateBuildOptions(meta, c.opts, d)
	if err != nil {
		return builder.InsertAdvancedQueryResult{}, createBuildResolved{}, nil, nil, err
	}

	buildRes, err := builder.BuildInsertQueryWithOptions(
		meta,
		d,
		c.orm.config,
		c.orm.placeholderMode(),
		resolved.BuildOpts,
	)
	if err != nil {
		return builder.InsertAdvancedQueryResult{}, createBuildResolved{}, nil, nil, err
	}

	return buildRes, resolved, meta, d, nil
}

func (c *CreateCommand) DryRun() (builder.DryRunResult, error) {
	if c == nil || c.orm == nil {
		return builder.DryRunResult{}, dictionary.ErrDBQueryEmpty
	}

	buildRes, _, _, d, err := c.prepareCreateWith()
	if err != nil {
		return builder.DryRunResult{}, err
	}

	res := builder.DryRunResult{
		Query: buildRes.Query,
		Args:  buildRes.Args,
		Mode:  buildRes.Mode,
	}

	c.orm.logDryRun(res.Query, d, buildRes.FilteredCols, res.Args, res.Mode)

	return res, nil
}

func (c *CreateCommand) exec(expectReturning bool) error {
	var (
		err   error
		start = time.Now()
	)

	if c == nil || c.orm == nil {
		return dictionary.ErrDBQueryEmpty
	}

	if err := c.validateReturningMode(expectReturning); err != nil {
		return err
	}

	if expectReturning {
		if err := c.resolveScanDest(); err != nil {
			return err
		}
	}

	buildRes, resolved, meta, d, err := c.prepareCreateWith()
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

	return executeMySQLAdvInsertScan(ctx, orm, meta, buildRes, resolved, dest)
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

	return executeOracleAdvInsertScan(ctx, orm, buildRes, dest)
}

func validateScanInto(dest []any) error {
	errs := errorpackage.Errors{}

	for i, d := range dest {
		if d == nil {
			// errs[rowKey(i)+":nil"] = dictionary.ErrDBScanNilDest
			errs.Add(rowKey(i)+":nil", dictionary.ErrDBScanNilDest)
			continue
		}

		rv := reflect.ValueOf(d)
		if rv.Kind() != reflect.Pointer || rv.IsNil() {
			// errs[rowKey(i)+":not_pointer"] = dictionary.ErrDBScanNotPointerDest
			errs.Add(rowKey(i)+":not_pointer", dictionary.ErrDBScanNotPointerDest)
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
