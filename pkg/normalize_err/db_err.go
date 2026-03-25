package normalizeerr

import (
	"strings"

	"github.com/siti-nabila/orm/pkg/dictionary"
)

type (
	DBError struct {
		Kind       error
		Dialect    string
		Constraint string
		Raw        error
	}
)

func (e *DBError) Error() string {
	if e.Raw != nil {
		return e.Raw.Error()
	}
	if e.Kind != nil {
		return e.Kind.Error()
	}
	return "unknown database error"
}

func (e *DBError) Unwrap() error {
	return e.Kind
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
			Kind:    dictionary.ErrDBUnknown,
			Dialect: dialect,
			Raw:     err,
		}
	}
}
