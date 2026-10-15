package orm_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/config"

	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/orm"
)

type resultInsert struct {
	ID   uint64  `sql:"column:id;primaryKey"`
	Flag int     `sql:"column:flag"`
	Note *string `sql:"column:note"`
}

func (resultInsert) TableName() string { return "result_inserts" }

type resultExecutor struct {
	d      dialect.Dialector
	ctx    context.Context
	query  string
	args   []any
	result driver.Result
	err    error
	calls  int
}

func (e *resultExecutor) Connect(context.Context) (driver.Conn, error) { return &resultConn{e}, nil }
func (e *resultExecutor) Driver() driver.Driver                        { return resultDriver{e} }

type resultDriver struct{ e *resultExecutor }

func (d resultDriver) Open(string) (driver.Conn, error) { return &resultConn{d.e}, nil }

type resultConn struct{ e *resultExecutor }

func (c *resultConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (c *resultConn) Close() error                        { return nil }
func (c *resultConn) Begin() (driver.Tx, error)           { return resultTx{}, nil }
func (c *resultConn) ExecContext(ctx context.Context, q string, args []driver.NamedValue) (driver.Result, error) {
	values := make([]any, len(args))
	for i, a := range args {
		values[i] = a.Value
	}
	return c.e.ExecContext(ctx, q, values...)
}

type resultTx struct{}

func (resultTx) Commit() error   { return nil }
func (resultTx) Rollback() error { return nil }
func resultAdapter(t *testing.T, e *resultExecutor, ctx context.Context) *orm.SqlTransactionAdapter {
	t.Helper()
	conn := sql.OpenDB(e)
	t.Cleanup(func() { conn.Close() })
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tx.Rollback() })
	return orm.NewSqlTransactionAdapter(ctx, tx, e.d, config.Config{UseSnakeCase: true})
}
func (e *resultExecutor) ExecContext(ctx context.Context, q string, args ...any) (driver.Result, error) {
	e.calls++
	e.ctx = ctx
	e.query = q
	e.args = args
	return e.result, e.err
}

type resultFailure struct{ err error }

func (r resultFailure) RowsAffected() (int64, error) { return 0, r.err }
func (r resultFailure) LastInsertId() (int64, error) { panic("must not assign generated ID") }

func TestCreateWithExecResultPreservesDryRunDriverResultAndContext(t *testing.T) {
	cause := errors.New("controlled result failure")
	for _, d := range []dialect.Dialector{dialect.NewPostgres(), dialect.NewMysql(), dialect.NewOracle()} {
		t.Run(d.Name(), func(t *testing.T) {
			for _, result := range []driver.Result{driver.RowsAffected(0), driver.RowsAffected(1), driver.RowsAffected(2), driver.RowsAffected(-1), resultFailure{cause}} {
				e := &resultExecutor{d: d, result: result}

				ctx := context.WithValue(context.Background(), struct{}{}, "request")
				o := resultAdapter(t, e, ctx)
				row := &resultInsert{}
				cmd := o.CreateWith(row)
				dry, err := cmd.DryRun()
				if err != nil || dry.Mode != builder.DryRunModeExec || strings.Contains(dry.Query, "RETURNING") {
					t.Fatal(dry, err)
				}
				got, err := cmd.ExecResult()
				if err != nil || got == nil || e.ctx != ctx || e.calls != 1 || e.query != dry.Query || !reflect.DeepEqual(e.args, []any{int64(0), nil}) || len(dry.Args) != 2 || row.ID != 0 {
					t.Fatal(got, err, e)
				}
				if result != nil {
					a, ae := result.RowsAffected()
					b, be := got.RowsAffected()
					if a != b || !errors.Is(ae, be) {
						t.Fatal(a, b, ae, be)
					}
				}
			}
			e := &resultExecutor{d: d, err: cause}
			o := resultAdapter(t, e, context.Background())
			if _, err := o.CreateWith(&resultInsert{}).ExecResult(); !errors.Is(err, cause) {
				t.Fatal(err)
			}
			calls := e.calls
			if _, err := o.CreateWith(&resultInsert{}).WithReturning("id").ExecResult(); err == nil || e.calls != calls {
				t.Fatal("returning executed", err)
			}
			if _, err := o.CreateWith(nil).ExecResult(); err == nil || e.calls != calls {
				t.Fatal("invalid model executed", err)
			}
		})
	}
}

func TestExecResultCancellationAndNilCommand(t *testing.T) {
	var nilCommand *orm.CreateCommand
	if _, err := nilCommand.ExecResult(); err == nil {
		t.Fatal("nil command accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	e := &resultExecutor{d: dialect.NewPostgres(), result: driver.RowsAffected(1)}
	tx := resultAdapter(t, e, ctx)
	cancel()
	if _, err := tx.CreateWith(&resultInsert{}).ExecResult(); !errors.Is(err, context.Canceled) || e.calls != 0 {
		t.Fatal(err, e.calls)
	}
}
