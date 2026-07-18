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
