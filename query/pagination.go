package query

import (
	"context"
	"fmt"
	"reflect"
	"strings"

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

func (b *QueryBuilder) ScanPaginate(ctx context.Context, dest any, opts pagination.PaginationOptions) (*pagination.PageMeta, error) {
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

	pageBuilder, err := b.scanPaginateBuilder(opts)
	if err != nil {
		return nil, err
	}
	dataRes, err := pageBuilder.build()
	if err != nil {
		return nil, err
	}

	if err := b.orm.ScanQuery(scanCtx, dataRes.Query, dataRes.Args, dataRes.SelectedCols, dest); err != nil {
		return nil, err
	}
	nextCursor := extractInMemoryNextCursor(dest, opts)
	applyInMemoryPaginationOffset(dest, opts)

	pageMeta := pagination.BuildPageMeta(opts, totalRows)
	pageMeta.NextCursor = nextCursor
	return &pageMeta, nil
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

	pageBuilder, err := b.scanPaginateBuilder(opts)
	if err != nil {
		return builder.ScanPaginateDryRunResult{}, err
	}

	dataRes, err := pageBuilder.build()
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

func (b *QueryBuilder) scanPaginateBuilder(opts pagination.PaginationOptions) (*QueryBuilder, error) {
	pageBuilder := b.clone()
	limit := opts.PerPage
	offset := pagination.OptionsOffset(opts)

	if opts.InMemoryOffset != nil {
		limit = opts.InMemoryOffset.MaxLimit
		pageBuilder.limit = &limit
		pageBuilder.offset = nil
		pageBuilder.singleRow = false
		return pageBuilder, nil
	}

	pageBuilder.limit = &limit
	pageBuilder.offset = &offset
	pageBuilder.singleRow = false
	return pageBuilder, nil
}

func applyInMemoryPaginationOffset(dest any, opts pagination.PaginationOptions) {
	if opts.InMemoryOffset == nil {
		return
	}

	slice := reflect.ValueOf(dest).Elem()
	offset := pagination.OptionsOffset(opts)
	if offset >= slice.Len() {
		slice.Set(reflect.MakeSlice(slice.Type(), 0, 0))
		return
	}

	end := offset + opts.PerPage
	if end > slice.Len() {
		end = slice.Len()
	}
	slice.Set(slice.Slice(offset, end))
}

func extractInMemoryNextCursor(dest any, opts pagination.PaginationOptions) string {
	if opts.InMemoryOffset == nil || opts.InMemoryOffset.CursorField == "" {
		return ""
	}

	slice := reflect.ValueOf(dest).Elem()
	if slice.Len() == 0 {
		return ""
	}

	item := slice.Index(slice.Len() - 1)
	for item.Kind() == reflect.Pointer {
		if item.IsNil() {
			return ""
		}
		item = item.Elem()
	}
	if item.Kind() != reflect.Struct {
		return ""
	}

	field := findCursorField(item, opts.InMemoryOffset.CursorField)
	if !field.IsValid() {
		return ""
	}
	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			return ""
		}
		field = field.Elem()
	}

	return fmt.Sprint(field.Interface())
}

func findCursorField(item reflect.Value, cursorField string) reflect.Value {
	itemType := item.Type()
	for i := 0; i < item.NumField(); i++ {
		structField := itemType.Field(i)
		if !structField.IsExported() {
			continue
		}
		if strings.EqualFold(structField.Name, cursorField) {
			return item.Field(i)
		}
		if sqlColumnName(structField.Tag.Get("sql")) == cursorField {
			return item.Field(i)
		}
	}
	return reflect.Value{}
}

func sqlColumnName(tag string) string {
	if tag == "" || tag == "-" {
		return ""
	}
	for _, part := range strings.Split(tag, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "column:") {
			return strings.TrimPrefix(part, "column:")
		}
	}
	return ""
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
