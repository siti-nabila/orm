package orm_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/orm"
)

const scanPaginateDriverName = "orm_scan_paginate_test"

var (
	registerScanPaginateDriver sync.Once
	scanPaginateStateMu        sync.Mutex
	scanPaginateState          *scanPaginateTestState
)

type scanPaginateTestState struct {
	mu        sync.Mutex
	totalRows int64
	dataCols  []string
	dataRows  [][]driver.Value
	queries   []scanPaginateQuery
}

type scanPaginateQuery struct {
	Query string
	Args  []driver.NamedValue
}

func (s *scanPaginateTestState) record(query string, args []driver.NamedValue) {
	s.mu.Lock()
	defer s.mu.Unlock()

	copiedArgs := append([]driver.NamedValue(nil), args...)
	s.queries = append(s.queries, scanPaginateQuery{
		Query: query,
		Args:  copiedArgs,
	})
}

func (s *scanPaginateTestState) snapshotQueries() []scanPaginateQuery {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]scanPaginateQuery(nil), s.queries...)
}

func (s *scanPaginateTestState) resetQueries() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.queries = nil
}

type scanPaginateDriver struct{}

func (scanPaginateDriver) Open(string) (driver.Conn, error) {
	scanPaginateStateMu.Lock()
	defer scanPaginateStateMu.Unlock()

	return &scanPaginateConn{state: scanPaginateState}, nil
}

type scanPaginateConn struct {
	state *scanPaginateTestState
}

func (*scanPaginateConn) Prepare(string) (driver.Stmt, error) {
	return nil, driver.ErrSkip
}

func (*scanPaginateConn) Close() error {
	return nil
}

func (*scanPaginateConn) Begin() (driver.Tx, error) {
	return nil, driver.ErrSkip
}

func (c *scanPaginateConn) QueryContext(
	_ context.Context,
	query string,
	args []driver.NamedValue,
) (driver.Rows, error) {
	c.state.record(query, args)

	if strings.Contains(query, "COUNT(*)") {
		return &scanPaginateRows{
			columns: []string{"count"},
			values:  [][]driver.Value{{c.state.totalRows}},
		}, nil
	}

	return &scanPaginateRows{
		columns: c.state.columns(),
		values:  c.state.dataRows,
	}, nil
}

func (s *scanPaginateTestState) columns() []string {
	if len(s.dataCols) > 0 {
		return s.dataCols
	}
	return []string{"id", "name"}
}

type scanPaginateRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (r *scanPaginateRows) Columns() []string {
	return r.columns
}

func (*scanPaginateRows) Close() error {
	return nil
}

func (r *scanPaginateRows) Next(values []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}

	copy(values, r.values[r.index])
	r.index++
	return nil
}

type scanPaginateUser struct {
	ID   int64  `sql:"column:id;primaryKey"`
	Name string `sql:"column:name"`
}

func (scanPaginateUser) TableName() string {
	return "users"
}

type scanPaginateAccount struct {
	ID       int64  `sql:"column:id;primaryKey"`
	Username string `sql:"column:username"`
	Email    string `sql:"column:email"`
}

func (scanPaginateAccount) TableName() string {
	return "users"
}

func openScanPaginateDB(t *testing.T, state *scanPaginateTestState) *sql.DB {
	t.Helper()

	registerScanPaginateDriver.Do(func() {
		sql.Register(scanPaginateDriverName, scanPaginateDriver{})
	})

	scanPaginateStateMu.Lock()
	scanPaginateState = state
	scanPaginateStateMu.Unlock()

	conn, err := sql.Open(scanPaginateDriverName, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		conn.Close()
	})

	return conn
}

func TestScanPaginateExecutesCountThenDataAndScansSlice(t *testing.T) {
	state := &scanPaginateTestState{
		totalRows: 3,
		dataRows: [][]driver.Value{
			{int64(3), "three"},
		},
	}
	conn := openScanPaginateDB(t, state)
	db := orm.NewSqlQueryAdapter(context.Background(), conn, dialect.NewPostgres(), config.Config{})

	var users []scanPaginateUser
	pageInfo, err := db.UseModel(scanPaginateUser{}).
		Where("active = ?", true).
		OrderBy("id ASC").
		ScanPaginate(context.Background(), &users, orm.PaginationOptions{
			Page:    2,
			PerPage: 2,
		})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(users, []scanPaginateUser{{ID: 3, Name: "three"}}) {
		t.Fatalf("unexpected users: %+v", users)
	}

	wantPageInfo := &orm.PageInfo{
		Page:       2,
		PerPage:    2,
		TotalRows:  3,
		TotalPages: 2,
		HasNext:    false,
		HasPrev:    true,
	}
	if !reflect.DeepEqual(pageInfo, wantPageInfo) {
		t.Fatalf("unexpected page info: got=%+v want=%+v", pageInfo, wantPageInfo)
	}

	queries := state.snapshotQueries()
	if len(queries) != 2 {
		t.Fatalf("expected count and data query, got %+v", queries)
	}
	if !strings.HasPrefix(queries[0].Query, "SELECT COUNT(*) FROM users WHERE active = $1") {
		t.Fatalf("unexpected count query: %s", queries[0].Query)
	}
	for _, forbidden := range []string{"ORDER BY", "LIMIT", "OFFSET"} {
		if strings.Contains(queries[0].Query, forbidden) {
			t.Fatalf("count query should not contain %q: %s", forbidden, queries[0].Query)
		}
	}
	if !strings.Contains(queries[1].Query, "ORDER BY id ASC") ||
		!strings.HasSuffix(queries[1].Query, " LIMIT 2 OFFSET 2") {
		t.Fatalf("unexpected data query: %s", queries[1].Query)
	}
	if !reflect.DeepEqual(namedValues(queries[0].Args), []any{true}) ||
		!reflect.DeepEqual(namedValues(queries[1].Args), []any{true}) {
		t.Fatalf("unexpected query args: count=%+v data=%+v", queries[0].Args, queries[1].Args)
	}
}

func TestScanPaginateSecondPageHasPrevAndNextWithExampleUserData(t *testing.T) {
	state := &scanPaginateTestState{
		totalRows: 23,
		dataCols:  []string{"id", "username", "email"},
		dataRows: [][]driver.Value{
			{int64(4), "user04", "user04@example.com"},
			{int64(5), "user05", "user05@example.com"},
			{int64(6), "user06", "user06@example.com"},
		},
	}
	conn := openScanPaginateDB(t, state)
	db := orm.NewSqlQueryAdapter(context.Background(), conn, dialect.NewPostgres(), config.Config{})

	var users []scanPaginateAccount
	pageInfo, err := db.UseModel(scanPaginateAccount{}).
		OrderBy("id ASC").
		ScanPaginate(context.Background(), &users, orm.PaginationOptions{
			Page:    2,
			PerPage: 3,
		})
	if err != nil {
		t.Fatal(err)
	}

	wantUsers := []scanPaginateAccount{
		{ID: 4, Username: "user04", Email: "user04@example.com"},
		{ID: 5, Username: "user05", Email: "user05@example.com"},
		{ID: 6, Username: "user06", Email: "user06@example.com"},
	}
	if !reflect.DeepEqual(users, wantUsers) {
		t.Fatalf("unexpected users: got=%+v want=%+v", users, wantUsers)
	}

	wantPageInfo := &orm.PageInfo{
		Page:       2,
		PerPage:    3,
		TotalRows:  23,
		TotalPages: 8,
		HasNext:    true,
		HasPrev:    true,
	}
	if !reflect.DeepEqual(pageInfo, wantPageInfo) {
		t.Fatalf("unexpected page info: got=%+v want=%+v", pageInfo, wantPageInfo)
	}

	queries := state.snapshotQueries()
	if len(queries) != 2 {
		t.Fatalf("expected count and data query, got %+v", queries)
	}
	if !strings.HasPrefix(queries[0].Query, "SELECT COUNT(*) FROM users") {
		t.Fatalf("unexpected count query: %s", queries[0].Query)
	}
	if !strings.HasSuffix(queries[1].Query, " ORDER BY id ASC LIMIT 3 OFFSET 3") {
		t.Fatalf("unexpected data query: %s", queries[1].Query)
	}
}

func TestScanPaginatePageInfoCalculation(t *testing.T) {
	tests := []struct {
		name     string
		total    int64
		page     int
		perPage  int
		wantInfo orm.PageInfo
	}{
		{
			name:    "zero rows on page greater than one",
			total:   0,
			page:    2,
			perPage: 10,
			wantInfo: orm.PageInfo{
				Page:    2,
				PerPage: 10,
			},
		},
		{
			name:    "exact division",
			total:   4,
			page:    1,
			perPage: 2,
			wantInfo: orm.PageInfo{
				Page:       1,
				PerPage:    2,
				TotalRows:  4,
				TotalPages: 2,
				HasNext:    true,
			},
		},
		{
			name:    "non exact division rounds up",
			total:   5,
			page:    1,
			perPage: 2,
			wantInfo: orm.PageInfo{
				Page:       1,
				PerPage:    2,
				TotalRows:  5,
				TotalPages: 3,
				HasNext:    true,
			},
		},
		{
			name:    "page greater than one has prev when rows exist",
			total:   5,
			page:    2,
			perPage: 2,
			wantInfo: orm.PageInfo{
				Page:       2,
				PerPage:    2,
				TotalRows:  5,
				TotalPages: 3,
				HasNext:    true,
				HasPrev:    true,
			},
		},
		{
			name:    "last page has no next",
			total:   3,
			page:    2,
			perPage: 2,
			wantInfo: orm.PageInfo{
				Page:       2,
				PerPage:    2,
				TotalRows:  3,
				TotalPages: 2,
				HasPrev:    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &scanPaginateTestState{totalRows: tt.total}
			conn := openScanPaginateDB(t, state)
			db := orm.NewSqlQueryAdapter(context.Background(), conn, dialect.NewPostgres(), config.Config{})

			var users []scanPaginateUser
			got, err := db.UseModel(scanPaginateUser{}).
				ScanPaginate(context.Background(), &users, orm.PaginationOptions{
					Page:    tt.page,
					PerPage: tt.perPage,
				})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, &tt.wantInfo) {
				t.Fatalf("unexpected page info: got=%+v want=%+v", got, &tt.wantInfo)
			}
		})
	}
}

func TestScanPaginateDoesNotChangeExistingScanAndFirst(t *testing.T) {
	state := &scanPaginateTestState{
		dataRows: [][]driver.Value{
			{int64(1), "one"},
			{int64(2), "two"},
		},
	}
	conn := openScanPaginateDB(t, state)
	db := orm.NewSqlQueryAdapter(context.Background(), conn, dialect.NewPostgres(), config.Config{})

	var users []scanPaginateUser
	if err := db.UseModel(scanPaginateUser{}).
		Where("active = ?", true).
		Scan(&users); err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 {
		t.Fatalf("unexpected scan users: %+v", users)
	}
	queries := state.snapshotQueries()
	if len(queries) != 1 || strings.Contains(queries[0].Query, "COUNT(*)") {
		t.Fatalf("scan should execute one non-count query, got %+v", queries)
	}

	state.resetQueries()
	state.dataRows = [][]driver.Value{{int64(1), "one"}}

	var user scanPaginateUser
	if err := db.UseModel(scanPaginateUser{}).
		Where("active = ?", true).
		First(&user); err != nil {
		t.Fatal(err)
	}
	if user.ID != 1 || user.Name != "one" {
		t.Fatalf("unexpected first user: %+v", user)
	}
	queries = state.snapshotQueries()
	if len(queries) != 1 ||
		strings.Contains(queries[0].Query, "COUNT(*)") ||
		!strings.HasSuffix(queries[0].Query, " LIMIT 1") {
		t.Fatalf("first should execute one limited non-count query, got %+v", queries)
	}
}

func namedValues(values []driver.NamedValue) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value.Value)
	}
	return out
}
