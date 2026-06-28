package ormpb

import (
	orm "github.com/siti-nabila/orm/orm"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

func QueryOptionsFromProto(in *QueryOptions) (orm.QueryOptions, error) {
	if in == nil {
		return orm.QueryOptions{}, nil
	}

	out := orm.QueryOptions{
		Page:   int(in.GetPage()),
		Limit:  int(in.GetLimit()),
		Select: append([]string(nil), in.GetSelect()...),
	}

	if inMemoryOffset := in.GetInMemoryOffset(); inMemoryOffset != nil {
		cursor := inMemoryOffset.GetCursor()
		out.InMemoryOffset = &orm.InMemoryOffsetOptions{
			MaxLimit: int(inMemoryOffset.GetMaxLimit()),
		}
		if cursor != nil {
			out.InMemoryOffset.Cursor = orm.Cursor{
				Field: cursor.GetField(),
				Value: cursor.GetValue(),
			}
		}
	}

	for _, sort := range in.GetSort() {
		if sort == nil {
			continue
		}

		out.Sort = append(out.Sort, orm.SortField{
			Field: sort.GetField(),
			Desc:  sort.GetDesc(),
		})
	}

	if search := in.GetSearch(); search != nil {
		mode, err := SearchModeToORM(search.GetMode())
		if err != nil {
			return orm.QueryOptions{}, err
		}

		out.Search = &orm.SearchQuery{
			Fields:  append([]string(nil), search.GetFields()...),
			Keyword: search.GetKeyword(),
			Mode:    mode,
		}
	}

	if searchAnd := in.GetSearchAnd(); searchAnd != nil {
		out.SearchAnd = &orm.SearchQueryAnd{}
		for _, field := range searchAnd.GetFields() {
			if field == nil {
				continue
			}

			out.SearchAnd.Fields = append(out.SearchAnd.Fields, &orm.SearchField{
				Field:   field.GetField(),
				Keyword: field.GetKeyword(),
			})
		}
	}

	for _, filter := range in.GetFilters() {
		if filter == nil {
			continue
		}

		op, err := FilterOperatorToORM(filter.GetOperator())
		if err != nil && len(filter.GetValues()) == 0 {
			return orm.QueryOptions{}, err
		}

		out.Filters = append(out.Filters, orm.Filter{
			Field:    filter.GetField(),
			Operator: op,
			Value:    filter.GetValue(),
			Values:   append([]string(nil), filter.GetValues()...),
		})
	}

	return out, nil
}

func (in *QueryOptions) ToORM() (orm.QueryOptions, error) {
	return QueryOptionsFromProto(in)
}

func SearchModeToORM(mode SearchMode) (orm.SearchMode, error) {
	switch mode {
	case SearchMode_SEARCH_MODE_UNSPECIFIED:
		return "", nil
	case SearchMode_SEARCH_MODE_CONTAINS:
		return orm.SearchModeContains, nil
	case SearchMode_SEARCH_MODE_PREFIX:
		return orm.SearchModePrefix, nil
	case SearchMode_SEARCH_MODE_FULL_TEXT:
		return orm.SearchModeFullText, nil
	case SearchMode_SEARCH_MODE_TRIGRAM:
		return orm.SearchModeTrigram, nil
	case SearchMode_SEARCH_MODE_FULL_TEXT_TRIGRAM:
		return orm.SearchModeFullTextTrigram, nil
	default:
		return "", dictionary.ErrInvalidSearchMode
	}
}

func FilterOperatorToORM(op FilterOperator) (orm.Operator, error) {
	switch op {
	case FilterOperator_FILTER_OPERATOR_EQUAL:
		return orm.OpEqual, nil
	case FilterOperator_FILTER_OPERATOR_NOT_EQUAL:
		return orm.OpNotEqual, nil
	case FilterOperator_FILTER_OPERATOR_GREATER_THAN:
		return orm.OpGreaterThan, nil
	case FilterOperator_FILTER_OPERATOR_GREATER_THAN_EQUAL:
		return orm.OpGreaterThanEqual, nil
	case FilterOperator_FILTER_OPERATOR_LESS_THAN:
		return orm.OpLessThan, nil
	case FilterOperator_FILTER_OPERATOR_LESS_THAN_EQUAL:
		return orm.OpLessThanEqual, nil
	case FilterOperator_FILTER_OPERATOR_LIKE:
		return orm.OpLike, nil
	case FilterOperator_FILTER_OPERATOR_NOT_LIKE:
		return orm.OpNotLike, nil
	default:
		return "", dictionary.ErrInvalidWhereOperator
	}
}
