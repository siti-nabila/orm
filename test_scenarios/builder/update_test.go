package builder_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

type taggedWhereUpdate struct {
	CompanyID   uint64  `sql:"column:company_id;where"`
	ReferenceID *string `sql:"column:reference_id;where"`
	Flag        int     `sql:"column:flag"`
	Active      bool    `sql:"column:active"`
	Note        string  `sql:"column:note"`
}

func (*taggedWhereUpdate) TableName() string { return "invoice_approval_logs" }

type primaryAndWhereUpdate struct {
	ID          uint64 `sql:"column:id;primaryKey"`
	ReferenceID string `sql:"column:reference_id;where"`
	Status      int    `sql:"column:status"`
}

func (*primaryAndWhereUpdate) TableName() string { return "invoices" }

type noUpdateWhere struct {
	Flag int `sql:"column:flag"`
}

func TestBuildUpdateQueryTaggedWhereStructAcrossDialects(t *testing.T) {
	ref := "INV-001"
	model := &taggedWhereUpdate{CompanyID: 0, ReferenceID: &ref, Flag: 0, Active: false, Note: ""}
	tests := []struct {
		name  string
		d     dialect.Dialector
		mode  config.PlaceholderMode
		query string
	}{
		{"postgres", dialect.NewPostgres(), config.PlaceholderByNumber, "UPDATE invoice_approval_logs SET flag = $1, active = $2, note = $3 WHERE company_id = $4 AND reference_id = $5"},
		{"mysql", dialect.NewMysql(), config.PlaceholderByNumber, "UPDATE invoice_approval_logs SET flag = ?, active = ?, note = ? WHERE company_id = ? AND reference_id = ?"},
		{"oracle", dialect.NewOracle(), config.PlaceholderByName, "UPDATE invoice_approval_logs SET flag = :flag, active = :active, note = :note WHERE company_id = :company_id AND reference_id = :reference_id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := builder.BuildUpdateQuery(model, tt.d, config.Config{UseSnakeCase: true}, tt.mode)
			if err != nil {
				t.Fatal(err)
			}
			if got.Query != tt.query {
				t.Fatalf("query\n got: %s\nwant: %s", got.Query, tt.query)
			}
			wantArgs := []any{0, false, "", uint64(0), &ref}
			if !reflect.DeepEqual(got.Args, wantArgs) {
				t.Fatalf("args: got %#v want %#v", got.Args, wantArgs)
			}
		})
	}
}

func TestBuildUpdateQueryPrimaryKeyPriority(t *testing.T) {
	got, err := builder.BuildUpdateQuery(
		&primaryAndWhereUpdate{ID: 10, ReferenceID: "INV-001", Status: 8},
		dialect.NewPostgres(), config.Config{UseSnakeCase: true}, config.PlaceholderByNumber,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Query != "UPDATE invoices SET status = $1 WHERE id = $2" {
		t.Fatalf("unexpected query: %s", got.Query)
	}
	if !reflect.DeepEqual(got.Args, []any{8, uint64(10)}) {
		t.Fatalf("unexpected args: %#v", got.Args)
	}

	_, err = builder.BuildUpdateQuery(
		&primaryAndWhereUpdate{ReferenceID: "INV-001", Status: 8},
		dialect.NewPostgres(), config.Config{UseSnakeCase: true}, config.PlaceholderByNumber,
	)
	if err == nil || !strings.Contains(err.Error(), dictionary.ErrPrimaryKeyEmpty.Error()) {
		t.Fatalf("expected ErrPrimaryKeyEmpty without where fallback, got %v", err)
	}
}

func TestBuildUpdateQueryPrimaryKeyMapBackwardCompatible(t *testing.T) {
	got, err := builder.BuildUpdateQuery(
		&primaryAndWhereUpdate{},
		dialect.NewPostgres(), config.Config{UseSnakeCase: true}, config.PlaceholderByNumber,
		map[string]any{"id": uint64(10), "status": 8},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Query != "UPDATE invoices SET status = $1 WHERE id = $2" {
		t.Fatalf("unexpected query: %s", got.Query)
	}
	if !reflect.DeepEqual(got.Args, []any{8, uint64(10)}) {
		t.Fatalf("unexpected args: %#v", got.Args)
	}
}

func TestBuildUpdateQueryTaggedWhereMap(t *testing.T) {
	ref := "INV-001"
	model := &taggedWhereUpdate{CompanyID: 7, ReferenceID: &ref}
	got, err := builder.BuildUpdateQuery(model, dialect.NewPostgres(), config.Config{UseSnakeCase: true}, config.PlaceholderByNumber, map[string]any{"flag": 0})
	if err != nil {
		t.Fatal(err)
	}
	if got.Query != "UPDATE invoice_approval_logs SET flag = $1 WHERE company_id = $2 AND reference_id = $3" {
		t.Fatalf("unexpected query: %s", got.Query)
	}
	if !reflect.DeepEqual(got.Args, []any{0, uint64(7), &ref}) {
		t.Fatalf("unexpected args: %#v", got.Args)
	}

	_, err = builder.BuildUpdateQuery(model, dialect.NewPostgres(), config.Config{}, config.PlaceholderByNumber, map[string]any{"reference_id": "other"})
	if err == nil || !strings.Contains(err.Error(), dictionary.ErrUpdateWhereFieldConflict.Error()) {
		t.Fatalf("expected where conflict, got %v", err)
	}
	_, err = builder.BuildUpdateQuery(model, dialect.NewPostgres(), config.Config{}, config.PlaceholderByNumber, map[string]any{"missing": 1})
	if err == nil || !strings.Contains(err.Error(), dictionary.ErrColumnNotFound.Error()) {
		t.Fatalf("expected column not found, got %v", err)
	}
	_, err = builder.BuildUpdateQuery(model, dialect.NewPostgres(), config.Config{}, config.PlaceholderByNumber, map[string]any{})
	if err == nil || !strings.Contains(err.Error(), dictionary.ErrDBQueryEmpty.Error()) {
		t.Fatalf("expected empty query, got %v", err)
	}
}

func TestBuildUpdateQueryWhereErrors(t *testing.T) {
	_, err := builder.BuildUpdateQuery(&taggedWhereUpdate{}, dialect.NewPostgres(), config.Config{}, config.PlaceholderByNumber)
	if err == nil || !strings.Contains(err.Error(), dictionary.ErrWhereValueNil.Error()) {
		t.Fatalf("expected nil tagged where error, got %v", err)
	}
	_, err = builder.BuildUpdateQuery(&noUpdateWhere{}, dialect.NewPostgres(), config.Config{}, config.PlaceholderByNumber)
	if err == nil || !strings.Contains(err.Error(), dictionary.ErrPrimaryKeyNotFound.Error()) {
		t.Fatalf("expected primary key not found, got %v", err)
	}
}
