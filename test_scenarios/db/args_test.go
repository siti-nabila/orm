package db_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"reflect"
	"testing"

	ormdb "github.com/siti-nabila/orm/db"
	"github.com/siti-nabila/orm/dialect"
)

type argumentCapture struct {
	calls [][]any
}

func (c *argumentCapture) record(args []driver.NamedValue) {
	values := make([]any, len(args))
	for i := range args {
		values[i] = args[i].Value
	}
	c.calls = append(c.calls, values)
}

func (c *argumentCapture) reset() {
	c.calls = nil
}

func (c *argumentCapture) last() []any {
	if len(c.calls) == 0 {
		return nil
	}
	return c.calls[len(c.calls)-1]
}

type captureConnector struct {
	capture *argumentCapture
}

func (c *captureConnector) Connect(context.Context) (driver.Conn, error) {
	return &captureConn{capture: c.capture}, nil
}

func (c *captureConnector) Driver() driver.Driver {
	return captureDriver{capture: c.capture}
}

type captureDriver struct {
	capture *argumentCapture
}

func (d captureDriver) Open(string) (driver.Conn, error) {
	return &captureConn{capture: d.capture}, nil
}

type captureConn struct {
	capture *argumentCapture
}

func (*captureConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported by the capture driver")
}

func (*captureConn) Close() error {
	return nil
}

func (*captureConn) Begin() (driver.Tx, error) {
	return captureTx{}, nil
}

func (*captureConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return captureTx{}, nil
}

func (c *captureConn) ExecContext(_ context.Context, _ string, args []driver.NamedValue) (driver.Result, error) {
	c.capture.record(args)
	return captureResult{}, nil
}

func (c *captureConn) QueryContext(_ context.Context, _ string, args []driver.NamedValue) (driver.Rows, error) {
	c.capture.record(args)
	return &captureRows{}, nil
}

type captureTx struct{}

func (captureTx) Commit() error   { return nil }
func (captureTx) Rollback() error { return nil }

type captureResult struct{}

func (captureResult) LastInsertId() (int64, error) { return 0, nil }
func (captureResult) RowsAffected() (int64, error) { return 1, nil }

type captureRows struct {
	read bool
}

func (*captureRows) Columns() []string { return []string{"value"} }
func (*captureRows) Close() error      { return nil }

func (r *captureRows) Next(dest []driver.Value) error {
	if r.read {
		return io.EOF
	}
	r.read = true
	dest[0] = int64(1)
	return nil
}

func openCaptureDatabase(t *testing.T) (*sql.DB, *argumentCapture) {
	t.Helper()

	capture := &argumentCapture{}
	conn := sql.OpenDB(&captureConnector{capture: capture})
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Errorf("close capture database: %v", err)
		}
	})
	return conn, capture
}

func exerciseExecutorMethods(t *testing.T, exec ormdb.Executor, capture *argumentCapture) {
	t.Helper()

	ctx := context.Background()
	methods := []struct {
		name string
		run  func() error
	}{
		{
			name: "exec",
			run: func() error {
				_, err := exec.Exec("SELECT $1", []int64{1, 2})
				return err
			},
		},
		{
			name: "query",
			run: func() error {
				rows, err := exec.Query("SELECT $1", []int64{1, 2})
				if err != nil {
					return err
				}
				return rows.Close()
			},
		},
		{
			name: "query_row",
			run: func() error {
				var value int64
				return exec.QueryRow("SELECT $1", []int64{1, 2}).Scan(&value)
			},
		},
		{
			name: "exec_context",
			run: func() error {
				_, err := exec.ExecContext(ctx, "SELECT $1", []int64{1, 2})
				return err
			},
		},
		{
			name: "query_context",
			run: func() error {
				rows, err := exec.QueryContext(ctx, "SELECT $1", []int64{1, 2})
				if err != nil {
					return err
				}
				return rows.Close()
			},
		},
		{
			name: "query_row_context",
			run: func() error {
				var value int64
				return exec.QueryRowContext(ctx, "SELECT $1", []int64{1, 2}).Scan(&value)
			},
		},
	}

	for _, method := range methods {
		t.Run(method.name, func(t *testing.T) {
			capture.reset()
			if err := method.run(); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(capture.last(), []any{"{1,2}"}) {
				t.Fatalf("expected PostgreSQL array arg, got %#v", capture.last())
			}
		})
	}
}

func TestPostgresArrayArgsAcrossDBMethods(t *testing.T) {
	conn, capture := openCaptureDatabase(t)
	exerciseExecutorMethods(t, ormdb.New(conn, dialect.NewPostgres()), capture)
}

func TestPostgresArrayArgsAcrossTransactionMethods(t *testing.T) {
	conn, capture := openCaptureDatabase(t)
	sqlTx, err := conn.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlTx.Rollback() })

	exerciseExecutorMethods(t, ormdb.NewTx(sqlTx, dialect.NewPostgres()), capture)
}

func TestPostgresArgNormalizationPreservesSpecialValues(t *testing.T) {
	conn, capture := openCaptureDatabase(t)
	exec := ormdb.New(conn, dialect.NewPostgres())

	var nilSlice *[]int64
	_, err := exec.ExecContext(
		context.Background(),
		"SELECT $1, $2, $3",
		nilSlice,
		[]byte("bytea"),
		"scalar",
	)
	if err != nil {
		t.Fatal(err)
	}

	want := []any{nil, []byte("bytea"), "scalar"}
	if !reflect.DeepEqual(capture.last(), want) {
		t.Fatalf("unexpected normalized args: got %#v want %#v", capture.last(), want)
	}
}

func TestNonPostgresArgsRemainUnchanged(t *testing.T) {
	for _, tt := range []struct {
		name string
		d    dialect.Dialector
	}{
		{name: "mysql", d: dialect.NewMysql()},
		{name: "oracle", d: dialect.NewOracle()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			conn, capture := openCaptureDatabase(t)
			exec := ormdb.New(conn, tt.d)

			_, err := exec.ExecContext(
				context.Background(),
				"SELECT ?, ?",
				[]byte("binary"),
				"scalar",
			)
			if err != nil {
				t.Fatal(err)
			}

			want := []any{[]byte("binary"), "scalar"}
			if !reflect.DeepEqual(capture.last(), want) {
				t.Fatalf("args changed: got %#v want %#v", capture.last(), want)
			}
		})
	}
}
