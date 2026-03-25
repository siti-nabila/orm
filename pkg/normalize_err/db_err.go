package normalizeerr

import (
	"strings"
)

const (
	KindUnknown      Kind = "unknown"
	KindDuplicateRow Kind = "duplicate_row"
	KindForeignKey   Kind = "foreign_key"
	KindRowNotFound  Kind = "row_not_found"
)

type (
	Kind    string
	DBError struct {
		Kind       Kind
		Dialect    string
		Constraint string
		Raw        error
	}
)

func (e *DBError) Error() string {
	if e.Raw != nil {
		return e.Raw.Error()
	}
	if e.Kind != "" {
		return string(e.Kind)
	}
	return "unknown database error"
}

func (e *DBError) Unwrap() error {
	return e.Raw
}

func Normalize(dialect string, err error) error {
	if err == nil {
		return nil
	}

	switch strings.ToLower(dialect) {
	case "postgres":
		return normalizePostgres(err)
	case "oracle":
		return normalizeOracle(err)
	case "mysql":
		return normalizeMySQL(err)
	default:
		return &DBError{
			Kind:    KindUnknown,
			Dialect: dialect,
			Raw:     err,
		}
	}
}
