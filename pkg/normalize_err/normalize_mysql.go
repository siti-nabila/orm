package normalizeerr

import (
	"errors"

	mysqlDriver "github.com/go-sql-driver/mysql"
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
				Kind:    KindDuplicateRow,
				Dialect: dialectName,
				Raw:     err,
			}

		case 1452, 1451: // cannot add/update child row: FK fails
			return &DBError{
				Kind:    KindForeignKey,
				Dialect: dialectName,
				Raw:     err,
			}

		}
	}

	return &DBError{
		Kind:    KindUnknown,
		Dialect: dialectName,
		Raw:     err,
	}
}
