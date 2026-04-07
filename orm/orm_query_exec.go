package orm

import (
	"github.com/siti-nabila/orm/pkg/logger"
	"github.com/siti-nabila/orm/query"
)

func (s *SqlQueryAdapter) UseModel(model any) *query.QueryBuilder {
	return s.orm.Q().WithContext(s.ctx).Table(model)
}

func (s *SqlQueryAdapter) SetLogger(l logger.Logger, debug bool) {
	s.orm.SetLogger(l, debug)
}
