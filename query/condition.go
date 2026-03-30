package query

const (
	ClauseAnd ClauseOperator = "AND"
	ClauseOr  ClauseOperator = "OR"
)

type (
	ClauseOperator string
	Condition      interface {
		isCondition()
	}

	ExpressionCondition struct {
		Operator ClauseOperator
		Query    string
		Args     []any
	}
	GroupCondition struct {
		Operator   ClauseOperator
		Conditions []Condition
	}
)

func (ExpressionCondition) isCondition() {}
func (GroupCondition) isCondition()      {}
