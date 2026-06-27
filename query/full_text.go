package query

import (
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

func (b *QueryBuilder) WhereFullText(column string, keyword string) (*QueryBuilder, error) {
	return b.whereFullText(ClauseAnd, column, keyword)
}

func (b *QueryBuilder) OrWhereFullText(column string, keyword string) (*QueryBuilder, error) {
	return b.whereFullText(ClauseOr, column, keyword)
}

func (b *QueryBuilder) whereFullText(
	clause ClauseOperator,
	column string,
	keyword string,
) (*QueryBuilder, error) {
	if b.orm == nil {
		return b, dictionary.ErrDBQueryEmpty
	}
	if column == "" {
		return b, nil
	}

	condition, err := dialect.BuildFullTextSearchCondition(
		b.orm.Dialect(),
		column,
		dialect.FullTextLanguageSimple,
	)
	if err != nil {
		return b, err
	}

	b.conditions = append(b.conditions, ExpressionCondition{
		Operator: clause,
		Query:    condition,
		Args:     []any{keyword},
	})

	return b, nil
}
