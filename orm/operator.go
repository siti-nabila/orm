package orm

import "github.com/siti-nabila/orm/query"

type Operator = query.Operator

const (
	OpEqual            = query.OpEqual
	OpNotEqual         = query.OpNotEqual
	OpLessThan         = query.OpLessThan
	OpLessThanEqual    = query.OpLessThanEqual
	OpGreaterThan      = query.OpGreaterThan
	OpGreaterThanEqual = query.OpGreaterThanEqual
	OpLike             = query.OpLike
	OpNotLike          = query.OpNotLike
)
