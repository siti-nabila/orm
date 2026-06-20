package query

import (
	"context"
	"reflect"

	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/pagination"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

const totalItemsColumn = "__orm_total_items"

func (b *QueryBuilder) Paginate(dest any, params pagination.Params) (pagination.Meta, error) {
	if b.orm == nil {
		return pagination.Meta{}, dictionary.ErrDBQueryEmpty
	}

	params, err := pagination.NormalizeWithConfig(params, b.orm.Config().Pagination)
	if err != nil {
		return pagination.Meta{}, err
	}
	if err := validatePaginationDest(dest); err != nil {
		return pagination.Meta{}, err
	}
	if b.withTotal && len(b.joins) > 0 {
		return pagination.Meta{}, dictionary.ErrPaginationTotalWithJoin
	}

	pageBuilder := b.paginationBuilder(params)
	res, err := pageBuilder.build()
	if err != nil {
		return pagination.Meta{}, err
	}

	var total *int64
	if b.withTotal {
		scanner, ok := b.orm.(pageQueryScanner)
		if !ok {
			return pagination.Meta{}, dictionary.ErrPaginationTotalUnsupported
		}
		total, err = scanner.ScanPageQuery(
			pageBuilder.ctx,
			res.Query,
			res.Args,
			res.SelectedCols,
			dest,
			totalItemsColumn,
		)
	} else {
		err = b.orm.ScanQuery(pageBuilder.ctx, res.Query, res.Args, res.SelectedCols, dest)
	}
	if err != nil {
		return pagination.Meta{}, err
	}

	itemCount := reflect.ValueOf(dest).Elem().Len()
	hasNext := false
	if !b.withTotal && itemCount > params.Limit {
		hasNext = true
		itemCount = params.Limit
		slice := reflect.ValueOf(dest).Elem()
		slice.Set(slice.Slice(0, itemCount))
	}

	return pagination.BuildMeta(params, itemCount, hasNext, total), nil
}

func (b *QueryBuilder) ScanPaginate(ctx context.Context, dest any, opts pagination.PaginationOptions) (*pagination.PageInfo, error) {
	if b.orm == nil {
		return nil, dictionary.ErrDBQueryEmpty
	}

	opts, err := pagination.NormalizeOptionsWithConfig(opts, b.orm.Config().Pagination)
	if err != nil {
		return nil, err
	}
	if err := validatePaginationDest(dest); err != nil {
		return nil, err
	}

	scanCtx := ctx
	if scanCtx == nil {
		scanCtx = b.ctx
	}

	countRes, err := b.buildCount()
	if err != nil {
		return nil, err
	}

	var totalRows int64
	if err := b.orm.ScanQuery(scanCtx, countRes.Query, countRes.Args, countRes.SelectedCols, &totalRows); err != nil {
		return nil, err
	}

	pageBuilder := b.scanPaginateBuilder(opts)
	dataRes, err := pageBuilder.build()
	if err != nil {
		return nil, err
	}

	if err := b.orm.ScanQuery(scanCtx, dataRes.Query, dataRes.Args, dataRes.SelectedCols, dest); err != nil {
		return nil, err
	}

	pageInfo := pagination.BuildPageInfo(opts, totalRows)
	return &pageInfo, nil
}

func (b *QueryBuilder) DryRunScanPaginate(opts pagination.PaginationOptions) (builder.ScanPaginateDryRunResult, error) {
	if b.orm == nil {
		return builder.ScanPaginateDryRunResult{}, dictionary.ErrDBQueryEmpty
	}

	opts, err := pagination.NormalizeOptionsWithConfig(opts, b.orm.Config().Pagination)
	if err != nil {
		return builder.ScanPaginateDryRunResult{}, err
	}

	countRes, err := b.buildCount()
	if err != nil {
		return builder.ScanPaginateDryRunResult{}, err
	}

	dataRes, err := b.scanPaginateBuilder(opts).build()
	if err != nil {
		return builder.ScanPaginateDryRunResult{}, err
	}

	return builder.ScanPaginateDryRunResult{
		Count: builder.DryRunResult{
			Query: countRes.Query,
			Args:  countRes.Args,
			Mode:  countRes.Mode,
		},
		Data: builder.DryRunResult{
			Query: dataRes.Query,
			Args:  dataRes.Args,
			Mode:  dataRes.Mode,
		},
	}, nil
}

func (b *QueryBuilder) DryRunPaginate(params pagination.Params) (builder.DryRunResult, error) {
	if b.orm == nil {
		return builder.DryRunResult{}, dictionary.ErrDBQueryEmpty
	}

	params, err := pagination.NormalizeWithConfig(params, b.orm.Config().Pagination)
	if err != nil {
		return builder.DryRunResult{}, err
	}
	if b.withTotal && len(b.joins) > 0 {
		return builder.DryRunResult{}, dictionary.ErrPaginationTotalWithJoin
	}

	res, err := b.paginationBuilder(params).build()
	if err != nil {
		return builder.DryRunResult{}, err
	}

	return builder.DryRunResult{
		Query: res.Query,
		Args:  res.Args,
		Mode:  res.Mode,
	}, nil
}

func (b *QueryBuilder) paginationBuilder(params pagination.Params) *QueryBuilder {
	pageBuilder := b.clone()
	limit := params.Limit
	if !b.withTotal {
		limit++
	}
	offset := pagination.Offset(params)
	pageBuilder.limit = &limit
	pageBuilder.offset = &offset
	pageBuilder.singleRow = false

	if b.withTotal {
		pageBuilder.includeModelColumns = true
		pageBuilder.selectExprs = append(
			pageBuilder.selectExprs,
			"COUNT(*) OVER() AS "+totalItemsColumn,
		)
	}

	return pageBuilder
}

func (b *QueryBuilder) scanPaginateBuilder(opts pagination.PaginationOptions) *QueryBuilder {
	pageBuilder := b.clone()
	limit := opts.PerPage
	offset := pagination.OptionsOffset(opts)
	pageBuilder.limit = &limit
	pageBuilder.offset = &offset
	pageBuilder.singleRow = false
	return pageBuilder
}

func validatePaginationDest(dest any) error {
	if dest == nil {
		return dictionary.ErrDBScanNilDest
	}
	rv := reflect.ValueOf(dest)
	if rv.Kind() != reflect.Ptr || rv.IsNil() || rv.Elem().Kind() != reflect.Slice {
		return dictionary.ErrMustBeSlicePtr
	}
	if rv.Elem().Type().Elem().Kind() != reflect.Struct {
		return dictionary.ErrDBScanMustBeSliceStruct
	}
	return nil
}
