package orm

import (
	"context"

	"github.com/siti-nabila/orm/pkg/dictionary"
	"github.com/siti-nabila/orm/query"
)

type (
	QueryBuilder = query.QueryBuilder

	QueryOptions struct {
		Page      int
		Limit     int
		Sort      []SortField
		Search    *SearchQuery
		SearchAnd *SearchQueryAnd
		Filters   []Filter
		Select    []string
	}

	SortField struct {
		Field string
		Desc  bool
	}

	SearchQuery struct {
		Fields  []string
		Keyword string
	}

	SearchField struct {
		Field   string
		Keyword string
	}

	SearchQueryAnd struct {
		Fields []*SearchField
	}

	Filter struct {
		Field    string
		Operator Operator
		Value    string
		Values   []string
	}
)

func QueryPage[T any](
	ctx context.Context,
	q *QueryBuilder,
	dest *[]T,
	allowedFields map[string]string,
	opts QueryOptions,
) (PageData[T], error) {
	if q == nil {
		return EmptyPageData[T](opts.Page, opts.Limit), dictionary.ErrDBQueryEmpty
	}
	if dest == nil {
		return EmptyPageData[T](opts.Page, opts.Limit), dictionary.ErrDBScanNilDest
	}

	pageBuilder, err := applyQueryOptions(q, allowedFields, opts)
	if err != nil {
		return EmptyPageData[T](opts.Page, opts.Limit), err
	}

	meta, err := pageBuilder.ScanPaginate(ctx, dest, PaginationOptions{
		Page:    opts.Page,
		PerPage: opts.Limit,
	})
	if err != nil {
		return EmptyPageData[T](opts.Page, opts.Limit), err
	}

	return NewPageData(*dest, *meta), nil
}

func applyQueryOptions(
	q *QueryBuilder,
	allowedFields map[string]string,
	opts QueryOptions,
) (*QueryBuilder, error) {
	columns, err := resolveAllowedColumns(allowedFields, opts.Select)
	if err != nil {
		return q, err
	}
	if len(columns) > 0 {
		q = q.Select(columns...)
	}

	for _, filter := range opts.Filters {
		column, err := resolveAllowedColumn(allowedFields, filter.Field)
		if err != nil {
			return q, err
		}

		if len(filter.Values) > 0 {
			q = q.WhereIn(column, filter.Values)
			continue
		}

		q, err = q.WhereOp(column, query.Operator(filter.Operator), filter.Value)
		if err != nil {
			return q, err
		}
	}

	if err := applySearch(q, allowedFields, opts.Search); err != nil {
		return q, err
	}
	if err := applySearchAnd(q, allowedFields, opts.SearchAnd); err != nil {
		return q, err
	}

	for _, sort := range opts.Sort {
		column, err := resolveAllowedColumn(allowedFields, sort.Field)
		if err != nil {
			return q, err
		}

		direction := "ASC"
		if sort.Desc {
			direction = "DESC"
		}
		q = q.OrderBy(column + " " + direction)
	}

	return q, nil
}

func applySearch(
	q *QueryBuilder,
	allowedFields map[string]string,
	search *SearchQuery,
) error {
	if search == nil || search.Keyword == "" || len(search.Fields) == 0 {
		return nil
	}

	columns, err := resolveAllowedColumns(allowedFields, search.Fields)
	if err != nil {
		return err
	}
	if len(columns) == 0 {
		return nil
	}

	keyword := "%" + search.Keyword + "%"
	q.WhereGroup(func(group *QueryBuilder) {
		for i, column := range columns {
			if i == 0 {
				group.WhereOp(column, query.OpLike, keyword)
				continue
			}
			group.OrWhereOp(column, query.OpLike, keyword)
		}
	})

	return nil
}

func applySearchAnd(
	q *QueryBuilder,
	allowedFields map[string]string,
	search *SearchQueryAnd,
) error {
	if search == nil || len(search.Fields) == 0 {
		return nil
	}

	for _, field := range search.Fields {
		if field == nil || field.Keyword == "" {
			continue
		}

		column, err := resolveAllowedColumn(allowedFields, field.Field)
		if err != nil {
			return err
		}
		if _, err := q.WhereOp(column, query.OpLike, "%"+field.Keyword+"%"); err != nil {
			return err
		}
	}

	return nil
}

func resolveAllowedColumns(allowedFields map[string]string, fields []string) ([]string, error) {
	columns := make([]string, 0, len(fields))
	for _, field := range fields {
		column, err := resolveAllowedColumn(allowedFields, field)
		if err != nil {
			return nil, err
		}
		columns = append(columns, column)
	}
	return columns, nil
}

func resolveAllowedColumn(allowedFields map[string]string, field string) (string, error) {
	column, ok := allowedFields[field]
	if !ok || column == "" {
		return "", dictionary.ErrColumnNotFound
	}
	return column, nil
}
