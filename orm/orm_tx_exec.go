package orm

import "github.com/siti-nabila/orm/pkg/logger"

func (s *SqlTransactionAdapter) Create(v any) error {
	return s.orm.Create(s.ctx, v)
}

func (s *SqlTransactionAdapter) Update(v any, fields ...map[string]any) error {
	return s.orm.Update(s.ctx, v, fields...)
}

func (s *SqlTransactionAdapter) Commit() error {
	return s.tx.Commit()
}

func (s *SqlTransactionAdapter) Rollback() error {
	return s.tx.Rollback()
}

func (s *SqlTransactionAdapter) SetLogger(l logger.Logger, debug bool) {
	s.orm.SetLogger(l, debug)
}

func (s *SqlTransactionAdapter) CreateBulk(v any) error {
	return s.orm.CreateBulk(s.ctx, v)
}
