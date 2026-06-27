package dialect

import (
	"fmt"

	"github.com/siti-nabila/orm/pkg/dictionary"
)

const (
	FullTextLanguageSimple  = "simple"
	FullTextLanguageEnglish = "english"
)

func BuildFullTextSearchCondition(d Dialector, column, language string) (string, error) {
	if d == nil || d.Type() != DialectPostgres {
		return "", dictionary.ErrUnsupportedSearchModeForDialect
	}
	if column == "" {
		return "", nil
	}
	if language == "" {
		language = FullTextLanguageSimple
	}

	return fmt.Sprintf("%s @@ websearch_to_tsquery('%s', ?)", column, language), nil
}
