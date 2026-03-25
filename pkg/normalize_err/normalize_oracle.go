package normalizeerr

import (
	"strings"

	"github.com/siti-nabila/orm/pkg/dictionary"
)

func normalizeOracle(err error) error {
	msg := err.Error()
	dialectName := "oracle"

	switch {
	case strings.Contains(msg, "ORA-00001"):
		return &DBError{
			Kind:    dictionary.ErrDuplicateRow,
			Dialect: dialectName,
			Raw:     err,
		}

	case strings.Contains(msg, "ORA-02291"), strings.Contains(msg, "ORA-02292"):
		return &DBError{
			Kind:    dictionary.ErrForeignKey,
			Dialect: dialectName,
			Raw:     err,
		}
	case strings.Contains(msg, "ORA-01403"), strings.Contains(msg, "ORA-01403"):
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
