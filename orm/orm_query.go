package orm

import (
	"github.com/siti-nabila/orm/query"
)

func (o *ORM) Q() *query.QueryBuilder {
	return query.New(o)
}
