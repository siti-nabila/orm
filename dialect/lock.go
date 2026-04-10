package dialect

import "context"

// const (
// 	LockResultBool LockResultType = iota
// 	LockResultOracle
// )

type (
	// LockResultType int
	LockDialect interface {
		TryLockQuery(ctx context.Context, key string) (query string, args []any, err error)
		ReleaseLockQuery(ctx context.Context, key string) (query string, args []any, needed bool, err error)
		Type() DialectType
	}
)
