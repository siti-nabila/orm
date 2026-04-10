package dialect

import (
	"context"
	"fmt"
)

type (
	Oracle struct{}
)

func NewOracle() Oracle {
	return Oracle{}
}

func (d Oracle) PlaceholderByNumber(n int) string {
	return fmt.Sprintf(":%d", n)
}
func (d Oracle) PlaceholderByName(s string) string {
	return ":" + s
}

func (d Oracle) QuoteIdentifier(s string) string {
	return `"` + s + `"`
}

func (d Oracle) SupportReturning() bool {
	return true
}

func (d Oracle) Name() string {
	return "oracle"
}

func (d Oracle) Type() DialectType {
	return DialectOracle
}

func (d Oracle) TryLockQuery(ctx context.Context, key string) (query string, args []any, err error) {

	query = `
	DECLARE
		v_handle VARCHAR2(128);
		BEGIN
		DBMS_LOCK.ALLOCATE_UNIQUE(lockname => :1, lockhandle => v_handle);
		:2 := DBMS_LOCK.REQUEST(
			lockhandle        => v_handle,
			lockmode          => DBMS_LOCK.X_MODE,
			timeout           => 0,
			release_on_commit => TRUE
		);
		END;
	`
	return query, []any{key}, nil
}

func (d Oracle) ReleaseLockQuery(ctx context.Context, key string) (query string, args []any, needed bool, err error) {
	return "", nil, false, nil
}
