package normalizeerr

import (
	"strings"
)

func normalizeOracle(err error) error {
	msg := err.Error()
	dialectName := "oracle"

	switch {
	case strings.Contains(msg, "ORA-00001"):
		return &DBError{
			Kind:    KindDuplicateRow,
			Dialect: dialectName,
			Raw:     err,
		}

	case strings.Contains(msg, "ORA-02291"), strings.Contains(msg, "ORA-02292"):
		return &DBError{
			Kind:    KindForeignKey,
			Dialect: dialectName,
			Raw:     err,
		}
	case strings.Contains(msg, "ORA-01403"), strings.Contains(msg, "ORA-01403"):
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
