package orm

import (
	"context"
	"time"

	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
	normalizeerr "github.com/siti-nabila/orm/pkg/normalize_err"
)

func (o *ORM) CreateBulk(ctx context.Context, values any) (err error) {
	var (
		start             = time.Now()
		insertQueryResult builder.InsertBulkQueryResult
		d                 = o.Dialect()
		mode              builder.DryRunMode
	)

	defer func() {
		o.log(
			insertQueryResult.Query,
			d,
			insertQueryResult.FilteredCols,
			insertQueryResult.Args,
			mode,
			start,
			err,
		)
	}()

	sliceVal, _, isPtrElem, err := validateBulkValues(values)
	if err != nil {
		return err
	}

	metas, err := parseBulkMetas(sliceVal, isPtrElem, o.config.UseSnakeCase)
	if err != nil {
		return err
	}

	layout, err := resolveBulkInsertLayout(metas)
	if err != nil {
		return err
	}

	insertQueryResult, err = builder.BuildInsertBulkQuery(
		metas,
		layout.Table,
		layout.FilteredCols,
		layout.PrimaryKeyColName,
		layout.PrimaryKeyColIndexes,
		d,
		o.config,
		o.placeholderMode(),
	)
	if err != nil {
		return err
	}

	if insertQueryResult.Query == "" {
		return dictionary.ErrDBQueryEmpty
	}

	switch d.Type() {
	case dialect.DialectPostgres:
		mode = builder.DryRunModeQuery
		return o.createBulkPostgres(ctx, metas, insertQueryResult)
	case dialect.DialectMySQL:
		mode = builder.DryRunModeExec
		return o.createBulkMySQL(ctx, insertQueryResult)
	case dialect.DialectOracle:
		mode = builder.DryRunModeExec
		return o.createBulkOracle(ctx, insertQueryResult)
	default:
		return dictionary.ErrUnsupportedDialect
	}
}

func (o *ORM) createBulkMySQL(
	ctx context.Context,
	result builder.InsertBulkQueryResult,
) error {
	if result.Query == "" {
		return dictionary.ErrDBQueryEmpty
	}

	_, err := o.executor.ExecContext(ctx, result.Query, result.Args...)
	if err != nil {
		return normalizeerr.Normalize(o.Dialect().Name(), err)
	}

	return nil
}

func (o *ORM) createBulkOracle(
	ctx context.Context,
	result builder.InsertBulkQueryResult,
) error {
	if result.Query == "" {
		return dictionary.ErrDBQueryEmpty
	}

	_, err := o.executor.ExecContext(ctx, result.Query, result.Args...)
	if err != nil {
		return normalizeerr.Normalize(o.Dialect().Name(), err)
	}

	return nil
}

func (o *ORM) createBulkPostgres(
	ctx context.Context,
	metas []*mapper.Meta,
	result builder.InsertBulkQueryResult,
) error {
	if result.Query == "" {
		return dictionary.ErrDBQueryEmpty
	}

	rows, err := o.executor.QueryContext(ctx, result.Query, result.Args...)
	if err != nil {
		return normalizeerr.Normalize(o.Dialect().Name(), err)
	}
	defer rows.Close()

	count := 0

	for rows.Next() {
		if count >= len(metas) {
			return dictionary.ErrBulkInsertColumnCountMismatch
		}

		pk := metas[count].GetPrimaryKeyColumn()
		if pk == nil {
			return dictionary.ErrBulkInsertPrimaryKeyMismatch
		}

		if err := rows.Scan(pk.FieldSrc.Addr().Interface()); err != nil {
			return normalizeerr.Normalize(o.Dialect().Name(), err)
		}

		count++
	}

	if err := rows.Err(); err != nil {
		return normalizeerr.Normalize(o.Dialect().Name(), err)
	}

	if count != len(metas) {
		return dictionary.ErrBulkInsertColumnCountMismatch
	}

	return nil
}
