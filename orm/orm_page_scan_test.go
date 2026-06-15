package orm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"sync"
	"testing"

	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/db"
	"github.com/siti-nabila/orm/dialect"
)

const pageScanTestDriverName = "orm_page_scan_test"

var registerPageScanDriver sync.Once

type pageScanTestDriver struct{}

func (pageScanTestDriver) Open(string) (driver.Conn, error) {
	return pageScanTestConn{}, nil
}

type pageScanTestConn struct{}

func (pageScanTestConn) Prepare(string) (driver.Stmt, error) {
	return nil, driver.ErrSkip
}

func (pageScanTestConn) Close() error {
	return nil
}

func (pageScanTestConn) Begin() (driver.Tx, error) {
	return nil, driver.ErrSkip
}

func (pageScanTestConn) QueryContext(
	context.Context,
	string,
	[]driver.NamedValue,
) (driver.Rows, error) {
	return &pageScanTestRows{}, nil
}

type pageScanTestRows struct {
	index int
}

func (*pageScanTestRows) Columns() []string {
	return []string{"id", "name", "__orm_total_items"}
}

func (*pageScanTestRows) Close() error {
	return nil
}

func (r *pageScanTestRows) Next(values []driver.Value) error {
	rows := [][]driver.Value{
		{int64(1), "one", int64(5)},
		{int64(2), "two", int64(5)},
	}
	if r.index >= len(rows) {
		return io.EOF
	}
	copy(values, rows[r.index])
	r.index++
	return nil
}

type pageScanTestModel struct {
	ID   int64  `sql:"column:id;primaryKey"`
	Name string `sql:"column:name"`
}

func TestScanPageQuery(t *testing.T) {
	registerPageScanDriver.Do(func() {
		sql.Register(pageScanTestDriverName, pageScanTestDriver{})
	})
	conn, err := sql.Open(pageScanTestDriverName, "")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	o := New(db.New(conn, dialect.NewPostgres()), config.Config{})
	var models []pageScanTestModel
	total, err := o.ScanPageQuery(
		context.Background(),
		"SELECT id, name, COUNT(*) OVER() AS __orm_total_items FROM users",
		nil,
		nil,
		&models,
		"__orm_total_items",
	)
	if err != nil {
		t.Fatal(err)
	}
	if total == nil || *total != 5 {
		t.Fatalf("unexpected total: %v", total)
	}
	if len(models) != 2 || models[0].ID != 1 || models[1].Name != "two" {
		t.Fatalf("unexpected models: %+v", models)
	}
}
