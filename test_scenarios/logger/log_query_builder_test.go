package logger_test

import (
	"strings"
	"testing"

	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/pkg/logger"
)

func TestInterpolatePostgresDereferencesScalarPointers(t *testing.T) {
	documentName := "invoice.pdf"
	documentPath := "/tmp/invoice.pdf"
	britradePath := "/tmp/britrade.pdf"

	got := logger.Interpolate(
		"UPDATE invoices SET document_name = $1, document_path = $2, document_path_britrade = $3 WHERE id = $4",
		dialect.NewPostgres(),
		nil,
		&documentName,
		&documentPath,
		&britradePath,
		667,
	)

	if strings.Contains(got, "0x") {
		t.Fatalf("expected pointer values to be dereferenced, got %q", got)
	}

	want := "UPDATE invoices SET document_name = 'invoice.pdf', document_path = '/tmp/invoice.pdf', document_path_britrade = '/tmp/britrade.pdf' WHERE id = 667"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestInterpolatePostgresNilScalarPointerAsNull(t *testing.T) {
	var documentName *string

	got := logger.Interpolate(
		"UPDATE invoices SET document_name = $1 WHERE id = $2",
		dialect.NewPostgres(),
		nil,
		documentName,
		667,
	)

	want := "UPDATE invoices SET document_name = NULL WHERE id = 667"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestInterpolateShortValuesRemainCompatibleAcrossDialects(t *testing.T) {
	tests := []struct {
		name  string
		query string
		d     dialect.Dialector
		want  string
	}{
		{name: "postgres", query: "SELECT $1", d: dialect.NewPostgres(), want: "SELECT 'O''Reilly'"},
		{name: "mysql", query: "SELECT ?", d: dialect.NewMysql(), want: "SELECT 'O''Reilly'"},
		{name: "oracle", query: "SELECT :1", d: dialect.NewOracle(), want: "SELECT 'O''Reilly'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := logger.Interpolate(tt.query, tt.d, nil, "O'Reilly")
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestInterpolateNumericArraysByDialect(t *testing.T) {
	tests := []struct {
		name  string
		query string
		d     dialect.Dialector
		want  string
	}{
		{name: "postgres", query: "SELECT $1", d: dialect.NewPostgres(), want: "SELECT ARRAY[1,2]"},
		{name: "mysql", query: "SELECT ?", d: dialect.NewMysql(), want: "SELECT (1,2)"},
		{name: "oracle", query: "SELECT :1", d: dialect.NewOracle(), want: "SELECT (1,2)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := logger.Interpolate(tt.query, tt.d, nil, []int32{1, 2})
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestInterpolateLongScalarTruncatesMiddle(t *testing.T) {
	value := strings.Repeat("a", 80) + "middle" + strings.Repeat("z", 80)
	got := logger.Interpolate("$1", dialect.NewPostgres(), nil, value)

	if len(got) != 120 {
		t.Fatalf("expected 120-byte rendered value, got %d", len(got))
	}
	if !strings.Contains(got, "...(truncated)...") {
		t.Fatalf("expected truncation marker in %q", got)
	}
	if !strings.HasPrefix(got, "'aaaa") || !strings.HasSuffix(got, "zzzz'") {
		t.Fatalf("expected beginning and ending to be preserved, got %q", got)
	}
}

func TestInterpolateLongPostgresArrayTruncatesMiddle(t *testing.T) {
	roles := make([]int64, 1000)
	for i := range roles {
		roles[i] = int64(17000 + i)
	}

	got := logger.Interpolate("$1", dialect.NewPostgres(), nil, roles)

	if len(got) > 120 {
		t.Fatalf("expected bounded array log, got %d bytes", len(got))
	}
	if !strings.HasPrefix(got, "ARRAY[17000") {
		t.Fatalf("expected array prefix to be preserved, got %q", got)
	}
	if !strings.Contains(got, "...(truncated)...") {
		t.Fatalf("expected truncation marker in %q", got)
	}
	if !strings.HasSuffix(got, "17999]") {
		t.Fatalf("expected array suffix to be preserved, got %q", got)
	}
	if strings.Contains(got, "17500") {
		t.Fatalf("expected middle values to be removed, got %q", got)
	}
}
