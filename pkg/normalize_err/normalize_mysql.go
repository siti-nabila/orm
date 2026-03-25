package normalizeerr

import (
	"errors"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

func normalizeMySQL(err error) error {
	var (
		myErr       *mysqlDriver.MySQLError
		dialectName = "mysql"
	)
	if errors.As(err, &myErr) {

		switch myErr.Number {

		case 1062: // duplicate entry
			return &DBError{
				Kind:    dictionary.ErrDuplicateRow,
				Dialect: dialectName,
				Raw:     err,
			}

		case 1452, 1451: // cannot add/update child row: FK fails
			return &DBError{
				Kind:    dictionary.ErrForeignKey,
				Dialect: dialectName,
				Raw:     err,
			}

		}
	}

	return &DBError{
		Kind:    dictionary.ErrDBUnknown,
		Dialect: dialectName,
		Raw:     err,
	}
}
