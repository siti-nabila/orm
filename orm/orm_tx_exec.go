package orm

import (
	"context"
	"time"

	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/lock"
	"github.com/siti-nabila/orm/pkg/dictionary"
	"github.com/siti-nabila/orm/pkg/logger"
)

const (
	LockAcquired    int64 = 1
	LockNotAcquired int64 = 0
)

func (s *SqlTransactionAdapter) Create(v any) error {
	return s.orm.Create(s.ctx, v)
}

func (s *SqlTransactionAdapter) Update(v any, fields ...map[string]any) error {
	return s.orm.Update(s.ctx, v, fields...)
}

func (s *SqlTransactionAdapter) Commit() error {
	if err := s.releaseLock(s.ctx); err != nil {
		return err
	}
	return s.tx.Commit()
}

func (s *SqlTransactionAdapter) Rollback() error {
	if err := s.releaseLock(s.ctx); err != nil {
		return err
	}

	return s.tx.Rollback()
}

func (s *SqlTransactionAdapter) SetLogger(l logger.Logger, debug bool) {
	s.orm.SetLogger(l, debug)
}

func (s *SqlTransactionAdapter) CreateBulk(v any) error {
	return s.orm.CreateBulk(s.ctx, v)
}

func (s *SqlTransactionAdapter) TryLock(ctx context.Context, key string) (bool, error) {

	var (
		start = time.Now()
		query string
		args  []any
		mode  builder.DryRunMode
	)
	normalizeKey := lock.Normalize(key)
	if normalizeKey == "" {
		return false, dictionary.ErrLockEmptyKey
	}

	if _, ok := s.acquiredLock[normalizeKey]; ok {
		return true, nil
	}

	d := s.orm.Dialect()
	lockDialect, ok := d.(dialect.LockDialect)
	if !ok {
		return false, dictionary.ErrLockUnsupportedDialect
	}

	query, args, err := lockDialect.TryLockQuery(ctx, normalizeKey)
	if err != nil {
		return false, err
	}

	if s.orm.shouldLogLockQuery() {
		defer func() {
			s.orm.log(
				query,
				d,
				nil,
				args,
				mode,
				start,
				err,
			)
		}()
	}
	var acq bool

	switch lockDialect.Type() {
	case dialect.DialectPostgres:
		mode = builder.DryRunModeQueryRow
		acq, err = s.tryLockBoolQuery(ctx, query, args)
	case dialect.DialectMySQL:
		mode = builder.DryRunModeQueryRow
		acq, err = s.tryLockMysql(ctx, query, args)
	case dialect.DialectOracle:
		mode = builder.DryRunModeExec
		acq, err = s.tryLockOracle(ctx, query, args)
	default:
		return false, dictionary.ErrLockUnsupportedDialect
	}

	if err != nil {
		return false, err
	}

	if acq {
		s.acquiredLock[normalizeKey] = struct{}{}

	}

	return acq, nil
}

func (s *SqlTransactionAdapter) CreateWith(v any) *CreateCommand {
	return &CreateCommand{
		orm: s.orm,
		ctx: s.ctx,
		v:   v,
	}
}
