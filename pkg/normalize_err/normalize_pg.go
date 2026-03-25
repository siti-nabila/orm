package normalizeerr

import (
	"database/sql"
	"errors"

	"github.com/lib/pq"
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
				Kind:       KindDuplicateRow,
				Dialect:    dialectName,
				Constraint: pqErr.Constraint,
				Raw:        err,
			}

		case "23503": // foreign key
			return &DBError{
				Kind:       KindForeignKey,
				Dialect:    dialectName,
				Constraint: pqErr.Constraint,
				Raw:        err,
			}
		}
	}

	if err == sql.ErrNoRows {
		return &DBError{
			Kind:    KindRowNotFound,
			Dialect: dialectName,
			Raw:     err,
		}
	}

	return &DBError{
		Kind:    KindUnknown,
		Dialect: dialectName,
		Raw:     err,
	}
}
