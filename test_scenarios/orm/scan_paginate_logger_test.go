package orm_test

import (
	"context"
	"database/sql/driver"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/orm"
)

type scanPaginateLogEntry struct {
	Query string
	Args  []any
	Mode  string
	Err   error
}

type scanPaginateLogger struct {
	entries []scanPaginateLogEntry
	dryRuns []scanPaginateLogEntry
}

func (l *scanPaginateLogger) Log(
	query string,
	_ dialect.Dialector,
	_ []mapper.ColumnMeta,
	args []any,
	mode string,
	_ time.Duration,
	err error,
) {
	l.entries = append(l.entries, scanPaginateLogEntry{
		Query: query,
		Args:  append([]any(nil), args...),
		Mode:  mode,
		Err:   err,
	})
}

func (l *scanPaginateLogger) LogDryRun(
	query string,
	_ dialect.Dialector,
	_ []mapper.ColumnMeta,
	args []any,
	mode string,
) {
	l.dryRuns = append(l.dryRuns, scanPaginateLogEntry{
		Query: query,
		Args:  append([]any(nil), args...),
		Mode:  mode,
	})
}

func TestScanPaginateLogsCountAndDataQueries(t *testing.T) {
	state := &scanPaginateTestState{
		totalRows: 2,
		dataRows: [][]driver.Value{
			{int64(1), "one"},
			{int64(2), "two"},
		},
	}
	conn := openScanPaginateDB(t, state)
	db := orm.NewSqlQueryAdapter(context.Background(), conn, dialect.NewPostgres(), config.Config{})

	log := &scanPaginateLogger{}
	db.SetLogger(log, true)

	var users []scanPaginateUser
	_, err := db.UseModel(scanPaginateUser{}).
		Where("active = ?", true).
		OrderBy("id ASC").
		ScanPaginate(context.Background(), &users, orm.PaginationOptions{
			Page:    1,
			PerPage: 2,
		})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(users, []scanPaginateUser{
		{ID: 1, Name: "one"},
		{ID: 2, Name: "two"},
	}) {
		t.Fatalf("unexpected users: %+v", users)
	}

	if len(log.entries) != 2 {
		t.Fatalf("expected count and data log entries, got %+v", log.entries)
	}
	if !strings.HasPrefix(log.entries[0].Query, "SELECT COUNT(*) FROM users WHERE active = $1") {
		t.Fatalf("unexpected count log query: %s", log.entries[0].Query)
	}
	if log.entries[0].Mode != builder.DryRunModeQueryRow.String() {
		t.Fatalf("unexpected count log mode: %s", log.entries[0].Mode)
	}
	if !reflect.DeepEqual(log.entries[0].Args, []any{true}) {
		t.Fatalf("unexpected count log args: %+v", log.entries[0].Args)
	}

	if !strings.Contains(log.entries[1].Query, "ORDER BY id ASC") ||
		!strings.HasSuffix(log.entries[1].Query, " LIMIT 2 OFFSET 0") {
		t.Fatalf("unexpected data log query: %s", log.entries[1].Query)
	}
	if log.entries[1].Mode != builder.DryRunModeQuery.String() {
		t.Fatalf("unexpected data log mode: %s", log.entries[1].Mode)
	}
	if !reflect.DeepEqual(log.entries[1].Args, []any{true}) {
		t.Fatalf("unexpected data log args: %+v", log.entries[1].Args)
	}
	if len(log.dryRuns) != 0 {
		t.Fatalf("ScanPaginate should not emit dry-run logs: %+v", log.dryRuns)
	}
}
