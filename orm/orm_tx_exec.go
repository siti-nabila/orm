package orm

import "github.com/siti-nabila/orm/pkg/logger"

func (s *SqlTransactionAdapter) Create(v any) error {
	return s.orm.Create(s.ctx, v)
}

func (s *SqlTransactionAdapter) Commit() error {
	return s.orm.Commit()
}

func (s *SqlTransactionAdapter) Rollback() error {
	return s.orm.Rollback()
}

func (s *SqlTransactionAdapter) SetLogger(l logger.Logger, debug bool) {
	s.orm.SetLogger(l, debug)
}
