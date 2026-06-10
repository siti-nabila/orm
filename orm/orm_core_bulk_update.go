package orm

import (
	"context"
	"time"

	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
	"github.com/siti-nabila/orm/pkg/helper"
	normalizeerr "github.com/siti-nabila/orm/pkg/normalize_err"
)

func (o *ORM) UpdateBulk(ctx context.Context, values any) (err error) {
	var (
		start            = time.Now()
		updateBulkResult builder.UpdateBulkQueryResult
		d                = o.Dialect()
		mode             builder.DryRunMode
	)

	defer func() {
		o.log(
			updateBulkResult.Query,
			d,
			updateBulkResult.FilteredCols,
			updateBulkResult.Args,
			mode,
			start,
			err,
		)
	}()

	updateBulkResult, err = o.prepareUpdateBulk(values)
	if err != nil {
		return err
	}

	mode = builder.DryRunModeExec

	_, err = o.executor.ExecContext(ctx, updateBulkResult.Query, updateBulkResult.Args...)
	if err != nil {
		return normalizeerr.Normalize(d.Name(), err)
	}

	return nil
}

func (o *ORM) DryRunUpdateBulk(values any) (builder.DryRunResult, error) {
	updateBulkResult, err := o.prepareUpdateBulk(values)
	if err != nil {
		return builder.DryRunResult{}, err
	}

	res := builder.DryRunResult{
		Query: updateBulkResult.Query,
		Args:  updateBulkResult.Args,
		Mode:  builder.DryRunModeExec,
	}

	o.logDryRun(
		res.Query,
		o.Dialect(),
		updateBulkResult.FilteredCols,
		res.Args,
		res.Mode,
	)

	return res, nil
}

func (o *ORM) prepareUpdateBulk(values any) (builder.UpdateBulkQueryResult, error) {
	sliceVal, _, isPtrElem, err := validateBulkValues(values)
	if err != nil {
		return builder.UpdateBulkQueryResult{}, err
	}

	metas, err := parseBulkMetas(sliceVal, isPtrElem, o.config.UseSnakeCase)
	if err != nil {
		return builder.UpdateBulkQueryResult{}, err
	}

	layout, err := resolveBulkUpdateLayout(metas)
	if err != nil {
		return builder.UpdateBulkQueryResult{}, err
	}

	result, err := builder.BuildUpdateBulkQuery(
		metas,
		layout.Table,
		layout.SetCols,
		layout.PKCol,
		o.Dialect(),
		o.config,
		o.placeholderMode(),
	)
	if err != nil {
		return builder.UpdateBulkQueryResult{}, err
	}

	if result.Query == "" {
		return builder.UpdateBulkQueryResult{}, dictionary.ErrDBQueryEmpty
	}

	return result, nil
}

type BulkUpdateLayout struct {
	Table   string
	SetCols []mapper.ColumnMeta
	PKCol   mapper.ColumnMeta
}

func resolveBulkUpdateLayout(metas []*mapper.Meta) (*BulkUpdateLayout, error) {
	if len(metas) == 0 {
		return nil, dictionary.ErrBulkUpdateEmptyMetas
	}

	first := metas[0]

	pk := first.GetPrimaryKeyColumn()
	if pk == nil {
		return nil, dictionary.ErrPrimaryKeyNotFound
	}

	if helper.IsZero(pk.Value) {
		return nil, dictionary.ErrPrimaryKeyEmpty
	}

	setCols := filterUpdateColumnsForBulk(first.Columns)
	if len(setCols) == 0 {
		return nil, dictionary.ErrDBQueryEmpty
	}

	layout := &BulkUpdateLayout{
		Table:   first.Table,
		SetCols: setCols,
		PKCol:   *pk,
	}

	for _, meta := range metas[1:] {
		if meta.Table != layout.Table {
			return nil, dictionary.ErrBulkInsertTableMismatch
		}

		metaPK := meta.GetPrimaryKeyColumn()
		if metaPK == nil {
			return nil, dictionary.ErrPrimaryKeyNotFound
		}

		if metaPK.Name != layout.PKCol.Name {
			return nil, dictionary.ErrBulkUpdatePrimaryKeyMismatch
		}

		if helper.IsZero(metaPK.Value) {
			return nil, dictionary.ErrPrimaryKeyEmpty
		}

		rowCols := filterUpdateColumnsForBulk(meta.Columns)
		if len(rowCols) != len(layout.SetCols) {
			return nil, dictionary.ErrBulkUpdateColumnMismatch
		}

		for i := range layout.SetCols {
			if layout.SetCols[i].Name != rowCols[i].Name {
				return nil, dictionary.ErrBulkUpdateColumnMismatch
			}
		}
	}

	return layout, nil
}

// filterUpdateColumnsForBulk returns non-primary-key columns for update SET clause.
func filterUpdateColumnsForBulk(cols []mapper.ColumnMeta) []mapper.ColumnMeta {
	out := make([]mapper.ColumnMeta, 0, len(cols))
	for _, c := range cols {
		if c.PrimaryKey {
			continue
		}
		out = append(out, c)
	}
	return out
}
