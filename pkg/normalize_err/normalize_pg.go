package normalizeerr

import (
	"database/sql"
	"errors"

	"github.com/lib/pq"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

func normalizePostgres(err error) error {

	var (
		pqErr       *pq.Error
		dialectName = "postgres"
	)
	if errors.As(err, &pqErr) {

		switch string(pqErr.Code) {

		case "23505": // duplicate
			return &DBError{
				Kind:       dictionary.ErrDuplicateRow,
				Dialect:    dialectName,
				Constraint: pqErr.Constraint,
				Raw:        err,
			}

		case "23503": // foreign key
			return &DBError{
				Kind:       dictionary.ErrForeignKey,
				Dialect:    dialectName,
				Constraint: pqErr.Constraint,
				Raw:        err,
			}
		}
	}

	if errors.Is(err, sql.ErrNoRows) {
		return &DBError{
			Kind:    dictionary.ErrRowNotFound,
			Dialect: dialectName,
			Raw:     err,
		}
	}

	return &DBError{
		Kind:    dictionary.ErrDBUnknown,
		Dialect: dialectName,
		Raw:     err,
	}
}
