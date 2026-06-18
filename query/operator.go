package query

import "github.com/siti-nabila/orm/pkg/dictionary"

type Operator string

const (
	OpEqual            Operator = "equal"
	OpNotEqual         Operator = "not_equal"
	OpLessThan         Operator = "less_than"
	OpLessThanEqual    Operator = "less_than_equal"
	OpGreaterThan      Operator = "greater_than"
	OpGreaterThanEqual Operator = "greater_than_equal"
	OpLike             Operator = "like"
	OpNotLike          Operator = "not_like"
)

var operatorSQL = map[Operator]string{
	OpEqual:            "=",
	OpNotEqual:         "<>",
	OpLessThan:         "<",
	OpLessThanEqual:    "<=",
	OpGreaterThan:      ">",
	OpGreaterThanEqual: ">=",
	OpLike:             "LIKE",
	OpNotLike:          "NOT LIKE",
}

func (op Operator) String() string {
	return string(op)
}

func (op Operator) SQL() (string, error) {
	sqlOperator, ok := operatorSQL[op]
	if !ok {
		return "", dictionary.ErrInvalidWhereOperator
	}

	return sqlOperator, nil
}

func (b *QueryBuilder) WhereOp(column string, operator Operator, value any) (*QueryBuilder, error) {
	return b.whereOp(ClauseAnd, column, operator, value)
}

func (b *QueryBuilder) OrWhereOp(column string, operator Operator, value any) (*QueryBuilder, error) {
	return b.whereOp(ClauseOr, column, operator, value)
}

func (b *QueryBuilder) whereOp(
	clause ClauseOperator,
	column string,
	operator Operator,
	value any,
) (*QueryBuilder, error) {
	sqlOperator, err := operator.SQL()
	if err != nil {
		return b, err
	}

	if column == "" {
		return b, nil
	}

	b.conditions = append(b.conditions, ExpressionCondition{
		Operator: clause,
		Query:    column + " " + sqlOperator + " ?",
		Args:     []any{value},
	})

	return b, nil
}
