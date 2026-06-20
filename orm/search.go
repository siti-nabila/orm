package orm

import "github.com/siti-nabila/orm/pkg/dictionary"

type (
	SearchMode string

	FullTextLanguage string

	FullTextOperator string

	SearchFieldConfig struct {
		Column           string
		FullTextColumn   string
		FullTextLanguage FullTextLanguage
		FullTextOperator FullTextOperator
		Modes            []SearchMode
	}

	QueryPageConfig struct {
		// AllowedFields returns the allowed frontend-to-database field mapping.
		//
		// The map key is the JSON/request field name used by the frontend.
		// The map value is the actual database column name used by the query builder.
		//
		// Example:
		//
		//	map[string]string{
		//		"joinDate": "join_date",
		//		"email":    "email",
		//	}
		//
		// QueryPage uses this mapping to safely resolve fields for filtering,
		// searching, search-and, sorting, and selecting columns. Fields that are
		// not present in this map are rejected; do not accept raw SQL from
		// frontend input.
		AllowedFields map[string]string
		SearchFields  map[string]SearchFieldConfig
	}
)

const (
	SearchModeContains        SearchMode = "contains"
	SearchModePrefix          SearchMode = "prefix"
	SearchModeFullText        SearchMode = "full_text"
	SearchModeTrigram         SearchMode = "trigram"
	SearchModeFullTextTrigram SearchMode = "full_text_trigram"
)

const (
	FullTextSimple  FullTextLanguage = "simple"
	FullTextEnglish FullTextLanguage = "english"
)

const (
	FullTextMatch       FullTextOperator = "match"
	FullTextContains    FullTextOperator = "contains"
	FullTextContainedBy FullTextOperator = "contained_by"
)

var fullTextLanguageSQL = map[FullTextLanguage]string{
	FullTextSimple:  "simple",
	FullTextEnglish: "english",
}

var fullTextOperatorSQL = map[FullTextOperator]string{
	FullTextMatch:       "@@",
	FullTextContains:    "@>",
	FullTextContainedBy: "<@",
}

func (mode SearchMode) String() string {
	return string(mode)
}

func (mode SearchMode) normalized() (SearchMode, error) {
	if mode == "" {
		return SearchModeContains, nil
	}

	switch mode {
	case SearchModeContains,
		SearchModePrefix,
		SearchModeFullText,
		SearchModeTrigram,
		SearchModeFullTextTrigram:
		return mode, nil
	default:
		return "", dictionary.ErrInvalidSearchMode
	}
}

func (lang FullTextLanguage) String() string {
	return string(lang)
}

func (lang FullTextLanguage) SQL() (string, error) {
	if lang == "" {
		lang = FullTextSimple
	}

	sqlLang, ok := fullTextLanguageSQL[lang]
	if !ok {
		return "", dictionary.ErrInvalidFullTextLanguage
	}

	return sqlLang, nil
}

func (op FullTextOperator) String() string {
	return string(op)
}

func (op FullTextOperator) SQL() (string, error) {
	if op == "" {
		op = FullTextMatch
	}

	sqlOp, ok := fullTextOperatorSQL[op]
	if !ok {
		return "", dictionary.ErrInvalidFullTextOperator
	}

	return sqlOp, nil
}

func isFullTextSearchMode(mode SearchMode) bool {
	return mode == SearchModeFullText || mode == SearchModeFullTextTrigram
}
