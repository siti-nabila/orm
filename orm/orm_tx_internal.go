package orm

import (
	"context"
	"database/sql"
	"time"

	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/pkg/dictionary"
	normalizeerr "github.com/siti-nabila/orm/pkg/normalize_err"
)

func (s *SqlTransactionAdapter) tryLockBoolQuery(ctx context.Context, query string, args []any) (bool, error) {
	var acq bool

	if err := s.tx.QueryRowContext(ctx, query, args...).Scan(&acq); err != nil {
		return false, normalizeerr.Normalize(s.orm.Dialect().Name(), err)
	}

	return acq, nil
}

func (s *SqlTransactionAdapter) tryLockMysql(ctx context.Context, query string, args []any) (bool, error) {
	var result sql.NullInt64

	if err := s.tx.QueryRowContext(ctx, query, args...).Scan(&result); err != nil {
		return false, normalizeerr.Normalize(s.orm.Dialect().Name(), err)
	}

	if !result.Valid {
		return false, dictionary.ErrLockNotAcquired
	}

	return result.Int64 == LockAcquired, nil
}

func (s *SqlTransactionAdapter) tryLockOracle(ctx context.Context, query string, args []any) (bool, error) {
	var result int64

	execArgs := make([]any, 0, len(args)+1)
	execArgs = append(execArgs, args...)
	execArgs = append(execArgs, sql.Out{Dest: &result})

	if _, err := s.tx.ExecContext(ctx, query, execArgs...); err != nil {
		return false, normalizeerr.Normalize(s.orm.Dialect().Name(), err)
	}

	switch result {
	case 0:
		return true, nil
	case 1:
		return false, nil
	default:
		return false, dictionary.ErrLockNotAcquired
	}
}

func (s *SqlTransactionAdapter) releaseLock(ctx context.Context) error {
	if len(s.acquiredLock) == 0 {
		return nil
	}

	d := s.orm.Dialect()
	lockDialect, ok := d.(dialect.LockDialect)
	if !ok {
		return dictionary.ErrLockUnsupportedDialect
	}

	for key := range s.acquiredLock {
		var (
			released sql.NullInt64
			start    = time.Now()
		)

		query, args, needed, err := lockDialect.ReleaseLockQuery(ctx, key)
		if err != nil {
			return err
		}

		if !needed {
			continue
		}

		execErr := s.tx.QueryRowContext(ctx, query, args...).Scan(&released)
		if execErr != nil {
			execErr = normalizeerr.Normalize(s.orm.Dialect().Name(), execErr)
		}

		if s.orm.shouldLogLockQuery() {
			s.orm.log(
				query,
				d,
				nil,
				args,
				builder.DryRunModeQueryRow,
				start,
				execErr,
			)
		}

		if execErr != nil {
			return execErr
		}
	}

	s.acquiredLock = make(map[string]struct{})
	return nil
}
