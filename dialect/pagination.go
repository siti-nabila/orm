package dialect

import "fmt"

func BuildPaginationClause(d Dialector, limit, offset *int) string {
	if paginationDialect, ok := d.(PaginationDialector); ok {
		return paginationDialect.PaginationClause(limit, offset)
	}
	return buildLimitOffsetClause(limit, offset)
}

func BuildCountWrapperQuery(_ Dialector, query string) string {
	return "SELECT COUNT(*) FROM (" + query + ") count_table"
}

func buildLimitOffsetClause(limit, offset *int) string {
	var clause string
	if limit != nil {
		clause = fmt.Sprintf(" LIMIT %d", *limit)
	}
	if offset != nil {
		clause += fmt.Sprintf(" OFFSET %d", *offset)
	}
	return clause
}
