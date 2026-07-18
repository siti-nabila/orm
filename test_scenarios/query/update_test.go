package query_test

import (
	"context"
	"database/sql"
	"reflect"
	"strings"
	"testing"

	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
	"github.com/siti-nabila/orm/query"
)

type updateModel struct {
	ID        uint64 `sql:"column:id;primaryKey"`
	Reference string `sql:"column:reference;where"`
	Flag      int    `sql:"column:flag"`
	Active    bool   `sql:"column:active"`
	Note      string `sql:"column:note"`
	Ignored   string
}

func (*updateModel) TableName() string { return "approval_logs" }

type whereOnlyUpdateModel struct {
	CompanyID uint64 `sql:"column:company_id;where"`
	Reference string `sql:"column:reference;where"`
	Flag      int    `sql:"column:flag"`
}

func (*whereOnlyUpdateModel) TableName() string { return "approval_logs" }

type noWhereModel struct {
	Flag int `sql:"column:flag"`
}

type noSetModel struct {
	ID uint64 `sql:"column:id;primaryKey"`
}

type fakeResult struct{}

func (fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (fakeResult) RowsAffected() (int64, error) { return 3, nil }

type fakeORM struct {
	dialect dialect.Dialector
	mode    config.PlaceholderMode
	result  builder.UpdateQueryResult
	calls   int
	err     error
}

type readOnlyFakeORM struct{}

func (*readOnlyFakeORM) Dialect() dialect.Dialector { return dialect.NewPostgres() }
func (*readOnlyFakeORM) Config() config.Config      { return config.Config{UseSnakeCase: true} }
func (*readOnlyFakeORM) PlaceholderMode() config.PlaceholderMode {
	return config.PlaceholderByNumber
}
func (*readOnlyFakeORM) ScanQuery(context.Context, string, []any, []mapper.ColumnMeta, any) error {
	return nil
}

func (f *fakeORM) Dialect() dialect.Dialector              { return f.dialect }
func (f *fakeORM) Config() config.Config                   { return config.Config{UseSnakeCase: true} }
func (f *fakeORM) PlaceholderMode() config.PlaceholderMode { return f.mode }
func (f *fakeORM) ScanQuery(context.Context, string, []any, []mapper.ColumnMeta, any) error {
	return nil
}
func (f *fakeORM) ExecUpdateQuery(_ context.Context, r builder.UpdateQueryResult) (sql.Result, error) {
	f.calls++
	f.result = r
	if f.err != nil {
		return nil, f.err
	}
	return fakeResult{}, nil
}

func newFakeBuilder(model any) (*query.QueryBuilder, *fakeORM) {
	f := &fakeORM{dialect: dialect.NewPostgres(), mode: config.PlaceholderByNumber}
	return query.New(f).Table(model), f
}

func TestDryRunUpdatesUsesTaggedWhereAndTaggedSetColumns(t *testing.T) {
	b, _ := newFakeBuilder(&whereOnlyUpdateModel{CompanyID: 0, Reference: "", Flag: 0})
	got, err := b.DryRunUpdates()
	if err != nil {
		t.Fatal(err)
	}
	if got.Query != "UPDATE approval_logs SET flag = $1 WHERE company_id = $2 AND reference = $3" {
		t.Fatalf("unexpected query: %s", got.Query)
	}
	if !reflect.DeepEqual(got.Args, []any{0, uint64(0), ""}) {
		t.Fatalf("unexpected args: %#v", got.Args)
	}
	if got.Mode != builder.DryRunModeExec {
		t.Fatalf("unexpected mode: %s", got.Mode)
	}
}

func TestUpdatesExplicitConditionIsOnlyWhereSource(t *testing.T) {
	model := &updateModel{ID: 99, Reference: "tagged", Flag: 0, Active: false, Note: ""}
	b, f := newFakeBuilder(model)
	res, err := b.Where("tenant_id = ?", 4).OrWhere("reference = ?", "explicit").Updates()
	if err != nil {
		t.Fatal(err)
	}
	if f.calls != 1 {
		t.Fatalf("expected one execution, got %d", f.calls)
	}
	if f.result.Query != "UPDATE approval_logs SET flag = $1, active = $2, note = $3 WHERE tenant_id = $4 OR reference = $5" {
		t.Fatalf("unexpected query: %s", f.result.Query)
	}
	if !reflect.DeepEqual(f.result.Args, []any{0, false, "", 4, "explicit"}) {
		t.Fatalf("unexpected args: %#v", f.result.Args)
	}
	rows, err := res.RowsAffected()
	if err != nil || rows != 3 {
		t.Fatalf("rows affected: %d, %v", rows, err)
	}
}

func TestUpdatesRejectsUnsupportedExecutor(t *testing.T) {
	b := query.New(&readOnlyFakeORM{}).Table(&whereOnlyUpdateModel{
		Reference: "INV-001",
		Flag:      1,
	})

	_, err := b.Updates()
	if err == nil || !strings.Contains(err.Error(), dictionary.ErrUpdateExecutorUnsupported.Error()) {
		t.Fatalf("expected unsupported update executor error, got %v", err)
	}
}

func TestDryRunUpdatesConditionVariants(t *testing.T) {
	tests := []struct {
		name  string
		apply func(*query.QueryBuilder)
		where string
		args  []any
	}{
		{"where_in", func(b *query.QueryBuilder) { b.WhereIn("id", []int{1, 2}) }, "id IN ($2, $3)", []any{0, 1, 2}},
		{"or_where_in", func(b *query.QueryBuilder) { b.Where("x = ?", 1).OrWhereIn("id", []int{2, 3}) }, "x = $2 OR id IN ($3, $4)", []any{0, 1, 2, 3}},
		{"where_not_in", func(b *query.QueryBuilder) { b.WhereNotIn("id", []int{1, 2}) }, "id NOT IN ($2, $3)", []any{0, 1, 2}},
		{"or_where_not_in", func(b *query.QueryBuilder) { b.Where("x = ?", 1).OrWhereNotIn("id", []int{2, 3}) }, "x = $2 OR id NOT IN ($3, $4)", []any{0, 1, 2, 3}},
		{"where_group", func(b *query.QueryBuilder) {
			b.WhereGroup(func(q *query.QueryBuilder) { q.Where("x = ?", 1).OrWhere("y = ?", 2) })
		}, "(x = $2 OR y = $3)", []any{0, 1, 2}},
		{"or_where_group", func(b *query.QueryBuilder) {
			b.Where("z = ?", 0).OrWhereGroup(func(q *query.QueryBuilder) { q.Where("x = ?", 1).Where("y = ?", 2) })
		}, "z = $2 OR (x = $3 AND y = $4)", []any{0, 0, 1, 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, _ := newFakeBuilder(&whereOnlyUpdateModel{Flag: 0})
			tt.apply(b)
			got, err := b.DryRunUpdates()
			if err != nil {
				t.Fatal(err)
			}
			want := "UPDATE approval_logs SET flag = $1 WHERE " + tt.where
			if got.Query != want || !reflect.DeepEqual(got.Args, tt.args) {
				t.Fatalf("got %q %#v, want %q %#v", got.Query, got.Args, want, tt.args)
			}
		})
	}
}

func TestDryRunUpdatesFallbackAndSafetyErrors(t *testing.T) {
	b, _ := newFakeBuilder(&updateModel{ID: 10, Reference: "ignored", Flag: 1})
	got, err := b.DryRunUpdates()
	if err != nil || got.Query != "UPDATE approval_logs SET flag = $1, active = $2, note = $3 WHERE id = $4" {
		t.Fatalf("primary fallback: %q %v", got.Query, err)
	}

	b, _ = newFakeBuilder(&updateModel{Reference: "must-not-fallback", Flag: 1})
	_, err = b.DryRunUpdates()
	if err == nil || !strings.Contains(err.Error(), dictionary.ErrPrimaryKeyEmpty.Error()) {
		t.Fatalf("expected empty primary key, got %v", err)
	}
	b, _ = newFakeBuilder(&noWhereModel{})
	_, err = b.DryRunUpdates()
	if err == nil || !strings.Contains(err.Error(), dictionary.ErrUpdateWithoutWhere.Error()) {
		t.Fatalf("expected unsafe update error, got %v", err)
	}
	b, _ = newFakeBuilder(&noSetModel{ID: 1})
	_, err = b.DryRunUpdates()
	if err == nil || !strings.Contains(err.Error(), dictionary.ErrDBQueryEmpty.Error()) {
		t.Fatalf("expected empty set error, got %v", err)
	}
	for _, bad := range []any{nil, updateModel{}, (*updateModel)(nil)} {
		b, _ = newFakeBuilder(bad)
		_, err = b.DryRunUpdates()
		if err == nil || !strings.Contains(err.Error(), dictionary.ErrMustBeStructPtr.Error()) {
			t.Fatalf("model %#v: expected pointer error, got %v", bad, err)
		}
	}
}

func TestDryRunUpdatesDialects(t *testing.T) {
	tests := []struct {
		name  string
		d     dialect.Dialector
		mode  config.PlaceholderMode
		query string
	}{
		{"postgres", dialect.NewPostgres(), config.PlaceholderByNumber, "UPDATE approval_logs SET flag = $1 WHERE company_id = $2 AND reference = $3"},
		{"mysql", dialect.NewMysql(), config.PlaceholderByNumber, "UPDATE approval_logs SET flag = ? WHERE company_id = ? AND reference = ?"},
		{"oracle", dialect.NewOracle(), config.PlaceholderByName, "UPDATE approval_logs SET flag = :flag WHERE company_id = :company_id AND reference = :reference"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeORM{dialect: tt.d, mode: tt.mode}
			got, err := query.New(f).Table(&whereOnlyUpdateModel{Reference: "INV", Flag: 0}).DryRunUpdates()
			if err != nil || got.Query != tt.query || !reflect.DeepEqual(got.Args, []any{0, uint64(0), "INV"}) {
				t.Fatalf("got %q %#v %v", got.Query, got.Args, err)
			}
		})
	}
}
