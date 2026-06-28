package orm

import (
	"context"
	"fmt"
	"strings"

	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/pagination"
	"github.com/siti-nabila/orm/pkg/dictionary"
	"github.com/siti-nabila/orm/query"
)

type (
	QueryBuilder = query.QueryBuilder

	QueryOptions struct {
		Page           int
		Limit          int
		InMemoryOffset *InMemoryOffsetOptions
		Sort           []SortField
		Search         *SearchQuery
		SearchAnd      *SearchQueryAnd
		Filters        []Filter
		Select         []string
	}

	InMemoryOffsetOptions struct {
		Cursor   Cursor
		MaxLimit int
	}

	Cursor struct {
		Field string
		Value any
	}

	SortField struct {
		Field string
		Desc  bool
	}

	SearchQuery struct {
		Fields  []string
		Keyword string
		Mode    SearchMode
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

// QueryPage applies frontend query options and returns a deterministic page.
//
// The allowedFields map is the allowed frontend-to-database field mapping.
// The map key is the JSON/request field name used by the frontend. The map
// value is the actual database column name used by the query builder.
//
// Example:
//
//	map[string]string{
//		"joinDate": "join_date",
//		"email":    "email",
//	}
//
// QueryPage uses this mapping to safely resolve fields for filtering,
// searching, search-and, sorting, and selecting columns. Fields that are not
// present in the map are rejected; do not accept raw SQL from frontend input.
func QueryPage[T any](
	ctx context.Context,
	q *QueryBuilder,
	dest *[]T,
	allowedFields map[string]string,
	opts QueryOptions,
) (PageData[T], error) {
	return QueryPageWithConfig(ctx, q, dest, QueryPageConfig{
		AllowedFields: allowedFields,
	}, opts)
}

// QueryPageWithConfig applies frontend query options using explicit mapping
// config and returns a deterministic page.
//
// QueryPageWithConfig uses QueryPageConfig.AllowedFields to safely resolve
// fields for filtering, searching, search-and, sorting, and selecting columns.
// Fields that are not present in the map are rejected; do not accept raw SQL
// from frontend input.
func QueryPageWithConfig[T any](
	ctx context.Context,
	q *QueryBuilder,
	dest *[]T,
	cfg QueryPageConfig,
	opts QueryOptions,
) (PageData[T], error) {
	if q == nil {
		return EmptyPageData[T](opts.Page, opts.Limit), dictionary.ErrDBQueryEmpty
	}
	if dest == nil {
		return EmptyPageData[T](opts.Page, opts.Limit), dictionary.ErrDBScanNilDest
	}

	pageBuilder, err := applyQueryOptions(q, cfg, opts)
	if err != nil {
		return EmptyPageData[T](opts.Page, opts.Limit), err
	}

	meta, err := pageBuilder.ScanPaginate(ctx, dest, PaginationOptions{
		Page:           opts.Page,
		PerPage:        opts.Limit,
		InMemoryOffset: queryPageInMemoryOffset(opts.InMemoryOffset),
	})
	if err != nil {
		return EmptyPageData[T](opts.Page, opts.Limit), err
	}

	return NewPageData(*dest, *meta), nil
}

func applyQueryOptions(
	q *QueryBuilder,
	cfg QueryPageConfig,
	opts QueryOptions,
) (*QueryBuilder, error) {
	columns, err := resolveAllowedColumns(cfg.AllowedFields, opts.Select)
	if err != nil {
		return q, err
	}
	if len(columns) > 0 {
		q = q.Select(columns...)
	}

	for _, filter := range opts.Filters {
		column, err := resolveAllowedColumn(cfg.AllowedFields, filter.Field)
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

	if err := applySearch(q, cfg, opts.Search); err != nil {
		return q, err
	}
	if err := applySearchAnd(q, cfg.AllowedFields, opts.SearchAnd); err != nil {
		return q, err
	}

	if err := applyInMemoryOffsetCursor(q, cfg.AllowedFields, opts); err != nil {
		return q, err
	}

	for _, sort := range opts.Sort {
		column, err := resolveAllowedColumn(cfg.AllowedFields, sort.Field)
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

func queryPageInMemoryOffset(opts *InMemoryOffsetOptions) *pagination.InMemoryOffsetOptions {
	if opts == nil {
		return nil
	}

	return &pagination.InMemoryOffsetOptions{
		MaxLimit: opts.MaxLimit,
	}
}

func applyInMemoryOffsetCursor(
	q *QueryBuilder,
	allowedFields map[string]string,
	opts QueryOptions,
) error {
	if opts.InMemoryOffset == nil {
		return nil
	}

	cursor := opts.InMemoryOffset.Cursor
	if cursor.Field == "" || cursor.Value == nil {
		return dictionary.ErrPaginationCursorRequired
	}
	if value, ok := cursor.Value.(string); ok && strings.TrimSpace(value) == "" {
		return dictionary.ErrPaginationCursorRequired
	}

	column, err := resolveAllowedColumn(allowedFields, cursor.Field)
	if err != nil {
		return err
	}

	operator := query.OpGreaterThan
	for _, sort := range opts.Sort {
		if sort.Field != cursor.Field {
			continue
		}
		if sort.Desc {
			operator = query.OpLessThan
		}
		break
	}

	_, err = q.WhereOp(column, operator, cursor.Value)
	return err
}

func applySearch(
	q *QueryBuilder,
	cfg QueryPageConfig,
	search *SearchQuery,
) error {
	if search == nil || search.Keyword == "" || len(search.Fields) == 0 {
		return nil
	}

	mode, err := search.Mode.normalized()
	if err != nil {
		return err
	}

	if err := validateSearchModeForDialect(mode, q.DialectType()); err != nil {
		return err
	}

	fieldConfigs := make([]resolvedSearchField, 0, len(search.Fields))
	for _, field := range search.Fields {
		fieldConfig, err := resolveSearchField(cfg, field, mode)
		if err != nil {
			return err
		}
		fieldConfigs = append(fieldConfigs, fieldConfig)
	}
	if len(fieldConfigs) == 0 {
		return nil
	}

	q.WhereGroup(func(group *QueryBuilder) {
		for i, fieldConfig := range fieldConfigs {
			condition, args := buildSearchCondition(fieldConfig, search.Keyword, mode)
			if i == 0 {
				group.Where(condition, args...)
				continue
			}
			group.OrWhere(condition, args...)
		}
	})

	return nil
}

type resolvedSearchField struct {
	Column           string
	FullTextColumn   string
	FullTextLanguage string
	FullTextOperator string
}

func validateSearchModeForDialect(mode SearchMode, dialectType dialect.DialectType) error {
	switch mode {
	case SearchModeContains, SearchModePrefix:
		return nil
	case SearchModeFullText, SearchModeTrigram, SearchModeFullTextTrigram:
		if dialectType != dialect.DialectPostgres {
			return dictionary.ErrUnsupportedSearchModeForDialect
		}
		return nil
	default:
		return dictionary.ErrInvalidSearchMode
	}
}

func resolveSearchField(
	cfg QueryPageConfig,
	field string,
	mode SearchMode,
) (resolvedSearchField, error) {
	if fieldConfig, ok := cfg.SearchFields[field]; ok {
		return resolveConfiguredSearchField(fieldConfig, mode)
	}

	column, err := resolveAllowedColumn(cfg.AllowedFields, field)
	if err != nil {
		if isPortableSearchMode(mode) {
			return resolvedSearchField{}, err
		}
		return resolvedSearchField{}, dictionary.ErrSearchFieldNotAllowed
	}

	if !isPortableSearchMode(mode) {
		return resolvedSearchField{}, dictionary.ErrSearchFieldNotAllowed
	}

	return resolvedSearchField{Column: column}, nil
}

func resolveConfiguredSearchField(
	cfg SearchFieldConfig,
	mode SearchMode,
) (resolvedSearchField, error) {
	if !isSearchModeAllowedForField(mode, cfg.Modes) {
		return resolvedSearchField{}, dictionary.ErrSearchModeNotAllowedForField
	}

	if cfg.FullTextOperator != "" && !isFullTextSearchMode(mode) {
		return resolvedSearchField{}, dictionary.ErrFullTextOperatorRequiresFullTextMode
	}

	if cfg.Column == "" {
		return resolvedSearchField{}, dictionary.ErrSearchFieldNotAllowed
	}

	out := resolvedSearchField{Column: cfg.Column}

	if isFullTextSearchMode(mode) {
		if cfg.FullTextColumn == "" {
			return resolvedSearchField{}, dictionary.ErrFullTextColumnRequired
		}

		lang, err := cfg.FullTextLanguage.SQL()
		if err != nil {
			return resolvedSearchField{}, err
		}

		op, err := cfg.FullTextOperator.SQL()
		if err != nil {
			return resolvedSearchField{}, err
		}

		out.FullTextColumn = cfg.FullTextColumn
		out.FullTextLanguage = lang
		out.FullTextOperator = op
	}

	return out, nil
}

func buildSearchCondition(field resolvedSearchField, keyword string, mode SearchMode) (string, []any) {
	switch mode {
	case SearchModePrefix:
		return field.Column + " LIKE ?", []any{keyword + "%"}
	case SearchModeFullText:
		return fullTextCondition(field), []any{keyword}
	case SearchModeTrigram:
		return field.Column + " ILIKE ?", []any{"%" + keyword + "%"}
	case SearchModeFullTextTrigram:
		return buildFullTextTrigramCondition(field, keyword)
	default:
		return field.Column + " LIKE ?", []any{"%" + keyword + "%"}
	}
}

func fullTextCondition(field resolvedSearchField) string {
	return fmt.Sprintf(
		"%s %s websearch_to_tsquery('%s', ?)",
		field.FullTextColumn,
		field.FullTextOperator,
		field.FullTextLanguage,
	)
}

func buildFullTextTrigramCondition(field resolvedSearchField, keyword string) (string, []any) {
	return fmt.Sprintf(
			"%s @@ to_tsquery('%s', ?) OR %s ILIKE '%%' || lower(?) || '%%'",
			field.FullTextColumn,
			field.FullTextLanguage,
			field.Column,
		),
		[]any{buildFullTextPrefixQuery(keyword), normalizeSearchKeyword(keyword)}
}

func buildFullTextPrefixQuery(keyword string) string {
	tokens := strings.Fields(keyword)
	if len(tokens) == 0 {
		return keyword
	}

	for i, token := range tokens {
		tokens[i] = token + ":*"
	}
	return strings.Join(tokens, " & ")
}

func normalizeSearchKeyword(keyword string) string {
	tokens := strings.Fields(keyword)
	if len(tokens) == 0 {
		return keyword
	}
	return strings.Join(tokens, " ")
}

func isPortableSearchMode(mode SearchMode) bool {
	return mode == SearchModeContains || mode == SearchModePrefix
}

func isSearchModeAllowedForField(mode SearchMode, modes []SearchMode) bool {
	if len(modes) == 0 {
		return isPortableSearchMode(mode)
	}

	for _, allowedMode := range modes {
		normalized, err := allowedMode.normalized()
		if err != nil {
			return false
		}
		if normalized == mode {
			return true
		}
	}
	return false
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
