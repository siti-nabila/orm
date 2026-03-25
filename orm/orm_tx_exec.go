package orm

func (s *SqlTransactionAdapter) Create(v any) error {
	return s.orm.Create(s.ctx, v)
}

func (s *SqlTransactionAdapter) Commit() error {
	return s.tx.Commit()
}

func (s *SqlTransactionAdapter) Rollback() error {
	return s.tx.Rollback()
}
