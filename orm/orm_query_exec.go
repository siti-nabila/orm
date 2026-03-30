package orm

import (
	"github.com/siti-nabila/orm/pkg/logger"
	"github.com/siti-nabila/orm/query"
)

// func (s *SqlQueryAdapter) DB() *sql.DB {
// 	return s.db
// }

//	func (s *SqlQueryAdapter) ORM() *ORM {
//		return s.orm
//	}
func (s *SqlQueryAdapter) UseModel(model any) *query.QueryBuilder {
	return s.orm.Q().WithContext(s.ctx).Table(model)
}

func (s *SqlQueryAdapter) SetLogger(l logger.Logger, debug bool) {
	s.orm.SetLogger(l, debug)
}
