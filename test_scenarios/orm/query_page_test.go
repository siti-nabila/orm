package orm_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	orm "github.com/siti-nabila/orm/orm"
	"github.com/siti-nabila/orm/pkg/dictionary"
	"github.com/siti-nabila/orm/query"
)

type queryPageUser struct {
	ID       int64  `sql:"column:id;primaryKey"`
	Name     string `sql:"column:name"`
	Email    string `sql:"column:email"`
	Status   string `sql:"column:status"`
	JoinDate string `sql:"column:join_date"`
}

func (queryPageUser) TableName() string {
	return "users"
}

type queryPageCall struct {
	Query string
	Args  []any
}

type queryPageORM struct {
	dialect dialect.Dialector
	total   int64
	rows    []queryPageUser
	calls   []queryPageCall
}

func (o *queryPageORM) Dialect() dialect.Dialector {
	if o.dialect == nil {
		return dialect.NewPostgres()
	}
	return o.dialect
}

func (o *queryPageORM) Config() config.Config {
	return config.Config{}
}

func (o *queryPageORM) PlaceholderMode() config.PlaceholderMode {
	return config.PlaceholderByNumber
}

func (o *queryPageORM) ScanQuery(
	_ context.Context,
	query string,
	args []any,
	_ []mapper.ColumnMeta,
	dest any,
) error {
	o.calls = append(o.calls, queryPageCall{
		Query: query,
		Args:  append([]any(nil), args...),
	})

	if total, ok := dest.(*int64); ok {
		*total = o.total
		return nil
	}

	rv := reflect.ValueOf(dest)
	if rv.Kind() == reflect.Pointer && rv.Elem().Kind() == reflect.Slice {
		rv.Elem().Set(reflect.ValueOf(append([]queryPageUser(nil), o.rows...)))
	}
	return nil
}

func TestQueryPageReturnsPageDataAndAppliesAllowedFields(t *testing.T) {
	fake := &queryPageORM{
		total: 55,
		rows: []queryPageUser{
			{
				ID:       1,
				Name:     "Nabila",
				Email:    "nabila@example.com",
				Status:   "ACTIVE",
				JoinDate: "2026-01-10",
			},
		},
	}
	var users []queryPageUser

	pageData, err := orm.QueryPage(
		context.Background(),
		query.New(fake).Table(queryPageUser{}),
		&users,
		queryPageAllowedFields(),
		orm.QueryOptions{
			Page:  2,
			Limit: 20,
			Select: []string{
				"id",
				"name",
				"email",
			},
			Filters: []orm.Filter{
				{Field: "status", Operator: orm.OpEqual, Value: "ACTIVE"},
				{Field: "joinDate", Operator: orm.OpGreaterThanEqual, Value: "2026-01-01"},
				{Field: "joinDate", Operator: orm.OpLessThan, Value: "2026-03-23"},
				{Field: "status", Values: []string{"ACTIVE", "PENDING"}},
			},
			Search: &orm.SearchQuery{
				Fields:  []string{"name", "email"},
				Keyword: "nabila",
			},
			SearchAnd: &orm.SearchQueryAnd{
				Fields: []*orm.SearchField{
					{Field: "status", Keyword: "ACTIVE"},
				},
			},
			Sort: []orm.SortField{
				{Field: "joinDate", Desc: true},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(pageData.Items, fake.rows) {
		t.Fatalf("unexpected items: got=%+v want=%+v", pageData.Items, fake.rows)
	}
	if pageData.Total != 55 || pageData.Page != 2 || pageData.Limit != 20 ||
		pageData.TotalPages != 3 || !pageData.HasNext || !pageData.HasPrev {
		t.Fatalf("unexpected page data: %+v", pageData)
	}
	if len(fake.calls) != 2 {
		t.Fatalf("expected count and data query, got %+v", fake.calls)
	}

	countQuery := fake.calls[0].Query
	for _, fragment := range []string{
		"SELECT COUNT(*) FROM users WHERE",
		"status = $1",
		"join_date >= $2",
		"join_date < $3",
		"status IN ($4, $5)",
		"(name LIKE $6 OR email LIKE $7)",
		"status LIKE $8",
	} {
		if !strings.Contains(countQuery, fragment) {
			t.Fatalf("missing count fragment %q in %s", fragment, countQuery)
		}
	}
	for _, forbidden := range []string{"ORDER BY", "LIMIT", "OFFSET"} {
		if strings.Contains(countQuery, forbidden) {
			t.Fatalf("count query should not contain %q: %s", forbidden, countQuery)
		}
	}

	dataQuery := fake.calls[1].Query
	for _, fragment := range []string{
		"SELECT id, name, email FROM users WHERE",
		"ORDER BY join_date DESC",
		"LIMIT 20 OFFSET 20",
	} {
		if !strings.Contains(dataQuery, fragment) {
			t.Fatalf("missing data fragment %q in %s", fragment, dataQuery)
		}
	}

	wantArgs := []any{
		"ACTIVE",
		"2026-01-01",
		"2026-03-23",
		"ACTIVE",
		"PENDING",
		"%nabila%",
		"%nabila%",
		"%ACTIVE%",
	}
	if !reflect.DeepEqual(fake.calls[0].Args, wantArgs) ||
		!reflect.DeepEqual(fake.calls[1].Args, wantArgs) {
		t.Fatalf("unexpected args: count=%+v data=%+v", fake.calls[0].Args, fake.calls[1].Args)
	}
}

func TestSearchModeDefaultAndInvalidMode(t *testing.T) {
	fake := &queryPageORM{}
	var users []queryPageUser

	_, err := orm.QueryPage(
		context.Background(),
		query.New(fake).Table(queryPageUser{}),
		&users,
		queryPageAllowedFields(),
		orm.QueryOptions{
			Page:  1,
			Limit: 20,
			Search: &orm.SearchQuery{
				Fields:  []string{"name"},
				Keyword: "nabila",
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(fake.calls[0].Query, "name LIKE $1") ||
		!reflect.DeepEqual(fake.calls[0].Args, []any{"%nabila%"}) {
		t.Fatalf("default search should be contains, query=%s args=%+v", fake.calls[0].Query, fake.calls[0].Args)
	}

	fake = &queryPageORM{}
	_, err = orm.QueryPage(
		context.Background(),
		query.New(fake).Table(queryPageUser{}),
		&users,
		queryPageAllowedFields(),
		orm.QueryOptions{
			Page:  1,
			Limit: 20,
			Search: &orm.SearchQuery{
				Fields:  []string{"name"},
				Keyword: "nabila",
				Mode:    orm.SearchMode("invalid"),
			},
		},
	)
	if !queryPageSameError(err, dictionary.ErrInvalidSearchMode) {
		t.Fatalf("expected ErrInvalidSearchMode, got %v", err)
	}
}

func TestFullTextLanguageAndOperatorSQL(t *testing.T) {
	lang, err := orm.FullTextLanguage("").SQL()
	if err != nil || lang != "simple" {
		t.Fatalf("unexpected default full-text language: lang=%s err=%v", lang, err)
	}

	lang, err = orm.FullTextEnglish.SQL()
	if err != nil || lang != "english" {
		t.Fatalf("unexpected full-text language: lang=%s err=%v", lang, err)
	}

	_, err = orm.FullTextLanguage("indonesian").SQL()
	if !queryPageSameError(err, dictionary.ErrInvalidFullTextLanguage) {
		t.Fatalf("expected ErrInvalidFullTextLanguage, got %v", err)
	}

	op, err := orm.FullTextOperator("").SQL()
	if err != nil || op != "@@" {
		t.Fatalf("unexpected default full-text operator: op=%s err=%v", op, err)
	}

	op, err = orm.FullTextContainedBy.SQL()
	if err != nil || op != "<@" {
		t.Fatalf("unexpected full-text operator: op=%s err=%v", op, err)
	}

	_, err = orm.FullTextOperator("invalid").SQL()
	if !queryPageSameError(err, dictionary.ErrInvalidFullTextOperator) {
		t.Fatalf("expected ErrInvalidFullTextOperator, got %v", err)
	}
}

func TestQueryPageWithConfigSearchModes(t *testing.T) {
	tests := []struct {
		name      string
		mode      orm.SearchMode
		keyword   string
		config    orm.SearchFieldConfig
		fragments []string
		args      []any
	}{
		{
			name:    "contains",
			mode:    orm.SearchModeContains,
			keyword: "nabila",
			config: orm.SearchFieldConfig{
				Column: "name",
				Modes:  []orm.SearchMode{orm.SearchModeContains},
			},
			fragments: []string{"name LIKE $1"},
			args:      []any{"%nabila%"},
		},
		{
			name:    "prefix",
			mode:    orm.SearchModePrefix,
			keyword: "nab",
			config: orm.SearchFieldConfig{
				Column: "name",
				Modes:  []orm.SearchMode{orm.SearchModePrefix},
			},
			fragments: []string{"name LIKE $1"},
			args:      []any{"nab%"},
		},
		{
			name:    "full text",
			mode:    orm.SearchModeFullText,
			keyword: "joko yono",
			config: orm.SearchFieldConfig{
				Column:         "name",
				FullTextColumn: "fts_keyword",
				Modes:          []orm.SearchMode{orm.SearchModeFullText},
			},
			fragments: []string{"fts_keyword @@ to_tsquery('simple', $1)"},
			args:      []any{"joko & yono"},
		},
		{
			name:    "trigram",
			mode:    orm.SearchModeTrigram,
			keyword: "joko",
			config: orm.SearchFieldConfig{
				Column: "name",
				Modes:  []orm.SearchMode{orm.SearchModeTrigram},
			},
			fragments: []string{"name ILIKE $1"},
			args:      []any{"%joko%"},
		},
		{
			name:    "full text trigram two tokens",
			mode:    orm.SearchModeFullTextTrigram,
			keyword: "joko y",
			config: orm.SearchFieldConfig{
				Column:           "name",
				FullTextColumn:   "fts_keyword",
				FullTextLanguage: orm.FullTextEnglish,
				Modes:            []orm.SearchMode{orm.SearchModeFullTextTrigram},
			},
			fragments: []string{"fts_keyword @@ to_tsquery('english', $1)", "name ILIKE $2"},
			args:      []any{"joko", "%y%"},
		},
		{
			name:    "full text trigram three tokens",
			mode:    orm.SearchModeFullTextTrigram,
			keyword: "joko yono wo",
			config: orm.SearchFieldConfig{
				Column:         "name",
				FullTextColumn: "fts_keyword",
				Modes:          []orm.SearchMode{orm.SearchModeFullTextTrigram},
			},
			fragments: []string{"fts_keyword @@ to_tsquery('simple', $1)", "name ILIKE $2"},
			args:      []any{"joko & yono", "%wo%"},
		},
		{
			name:    "full text trigram one token uses full text only",
			mode:    orm.SearchModeFullTextTrigram,
			keyword: "joko",
			config: orm.SearchFieldConfig{
				Column:         "name",
				FullTextColumn: "fts_keyword",
				Modes:          []orm.SearchMode{orm.SearchModeFullTextTrigram},
			},
			fragments: []string{"fts_keyword @@ to_tsquery('simple', $1)"},
			args:      []any{"joko"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &queryPageORM{}
			var users []queryPageUser

			_, err := orm.QueryPageWithConfig(
				context.Background(),
				query.New(fake).Table(queryPageUser{}),
				&users,
				orm.QueryPageConfig{
					AllowedFields: queryPageAllowedFields(),
					SearchFields: map[string]orm.SearchFieldConfig{
						"name": tt.config,
					},
				},
				orm.QueryOptions{
					Page:  1,
					Limit: 20,
					Search: &orm.SearchQuery{
						Fields:  []string{"name"},
						Keyword: tt.keyword,
						Mode:    tt.mode,
					},
				},
			)
			if err != nil {
				t.Fatal(err)
			}

			for _, fragment := range tt.fragments {
				if !strings.Contains(fake.calls[0].Query, fragment) {
					t.Fatalf("missing fragment %q in %s", fragment, fake.calls[0].Query)
				}
			}
			if !reflect.DeepEqual(fake.calls[0].Args, tt.args) ||
				!reflect.DeepEqual(fake.calls[1].Args, tt.args) {
				t.Fatalf("unexpected args: count=%+v data=%+v want=%+v", fake.calls[0].Args, fake.calls[1].Args, tt.args)
			}
		})
	}
}

func TestQueryPageSearchModeErrors(t *testing.T) {
	tests := []struct {
		name    string
		dialect dialect.Dialector
		config  orm.SearchFieldConfig
		mode    orm.SearchMode
		wantErr error
	}{
		{
			name:    "mysql full text unsupported",
			dialect: dialect.NewMysql(),
			config: orm.SearchFieldConfig{
				Column:         "name",
				FullTextColumn: "fts_keyword",
				Modes:          []orm.SearchMode{orm.SearchModeFullText},
			},
			mode:    orm.SearchModeFullText,
			wantErr: dictionary.ErrUnsupportedSearchModeForDialect,
		},
		{
			name:    "oracle trigram unsupported",
			dialect: dialect.NewOracle(),
			config: orm.SearchFieldConfig{
				Column: "name",
				Modes:  []orm.SearchMode{orm.SearchModeTrigram},
			},
			mode:    orm.SearchModeTrigram,
			wantErr: dictionary.ErrUnsupportedSearchModeForDialect,
		},
		{
			name: "full text column required",
			config: orm.SearchFieldConfig{
				Column: "name",
				Modes:  []orm.SearchMode{orm.SearchModeFullText},
			},
			mode:    orm.SearchModeFullText,
			wantErr: dictionary.ErrFullTextColumnRequired,
		},
		{
			name: "mode not allowed for field",
			config: orm.SearchFieldConfig{
				Column: "name",
				Modes:  []orm.SearchMode{orm.SearchModeContains},
			},
			mode:    orm.SearchModePrefix,
			wantErr: dictionary.ErrSearchModeNotAllowedForField,
		},
		{
			name: "invalid language",
			config: orm.SearchFieldConfig{
				Column:           "name",
				FullTextColumn:   "fts_keyword",
				FullTextLanguage: orm.FullTextLanguage("invalid"),
				Modes:            []orm.SearchMode{orm.SearchModeFullText},
			},
			mode:    orm.SearchModeFullText,
			wantErr: dictionary.ErrInvalidFullTextLanguage,
		},
		{
			name: "invalid full text operator",
			config: orm.SearchFieldConfig{
				Column:           "name",
				FullTextColumn:   "fts_keyword",
				FullTextOperator: orm.FullTextOperator("invalid"),
				Modes:            []orm.SearchMode{orm.SearchModeFullText},
			},
			mode:    orm.SearchModeFullText,
			wantErr: dictionary.ErrInvalidFullTextOperator,
		},
		{
			name: "full text operator outside full text mode",
			config: orm.SearchFieldConfig{
				Column:           "name",
				FullTextOperator: orm.FullTextMatch,
				Modes:            []orm.SearchMode{orm.SearchModeTrigram},
			},
			mode:    orm.SearchModeTrigram,
			wantErr: dictionary.ErrFullTextOperatorRequiresFullTextMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &queryPageORM{dialect: tt.dialect}
			var users []queryPageUser

			_, err := orm.QueryPageWithConfig(
				context.Background(),
				query.New(fake).Table(queryPageUser{}),
				&users,
				orm.QueryPageConfig{
					AllowedFields: queryPageAllowedFields(),
					SearchFields: map[string]orm.SearchFieldConfig{
						"name": tt.config,
					},
				},
				orm.QueryOptions{
					Page:  1,
					Limit: 20,
					Search: &orm.SearchQuery{
						Fields:  []string{"name"},
						Keyword: "joko",
						Mode:    tt.mode,
					},
				},
			)
			if !queryPageSameError(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestQueryPageAdvancedModeRequiresSearchFieldConfig(t *testing.T) {
	fake := &queryPageORM{}
	var users []queryPageUser

	_, err := orm.QueryPage(
		context.Background(),
		query.New(fake).Table(queryPageUser{}),
		&users,
		queryPageAllowedFields(),
		orm.QueryOptions{
			Page:  1,
			Limit: 20,
			Search: &orm.SearchQuery{
				Fields:  []string{"name"},
				Keyword: "joko",
				Mode:    orm.SearchModeFullText,
			},
		},
	)
	if !queryPageSameError(err, dictionary.ErrSearchFieldNotAllowed) {
		t.Fatalf("expected ErrSearchFieldNotAllowed, got %v", err)
	}
}

func TestQueryPageReturnsEmptyPageDataOnUnknownField(t *testing.T) {
	fake := &queryPageORM{}
	var users []queryPageUser

	pageData, err := orm.QueryPage(
		context.Background(),
		query.New(fake).Table(queryPageUser{}),
		&users,
		queryPageAllowedFields(),
		orm.QueryOptions{
			Page:  2,
			Limit: 20,
			Sort:  []orm.SortField{{Field: "unknown"}},
		},
	)
	if !queryPageSameError(err, dictionary.ErrColumnNotFound) {
		t.Fatalf("expected ErrColumnNotFound, got %v", err)
	}
	if pageData.Items == nil || len(pageData.Items) != 0 || pageData.Page != 2 ||
		pageData.Limit != 20 || pageData.Total != 0 || pageData.TotalPages != 0 ||
		pageData.HasNext || pageData.HasPrev {
		t.Fatalf("unexpected empty page data: %+v", pageData)
	}
}

func TestQueryPageReturnsEmptyPageDataOnInvalidOperator(t *testing.T) {
	fake := &queryPageORM{}
	var users []queryPageUser

	pageData, err := orm.QueryPage(
		context.Background(),
		query.New(fake).Table(queryPageUser{}),
		&users,
		queryPageAllowedFields(),
		orm.QueryOptions{
			Page:  1,
			Limit: 20,
			Filters: []orm.Filter{
				{Field: "status", Operator: orm.Operator("invalid"), Value: "ACTIVE"},
			},
		},
	)
	if !queryPageSameError(err, dictionary.ErrInvalidWhereOperator) {
		t.Fatalf("expected ErrInvalidWhereOperator, got %v", err)
	}
	if pageData.Items == nil || len(pageData.Items) != 0 || pageData.Page != 1 ||
		pageData.Limit != 20 || pageData.Total != 0 || pageData.TotalPages != 0 ||
		pageData.HasNext || pageData.HasPrev {
		t.Fatalf("unexpected empty page data: %+v", pageData)
	}
}

func queryPageAllowedFields() map[string]string {
	return map[string]string{
		"id":       "id",
		"name":     "name",
		"email":    "email",
		"status":   "status",
		"joinDate": "join_date",
	}
}

func queryPageSameError(got, want error) bool {
	if got == nil || want == nil {
		return got == want
	}
	return got.Error() == want.Error()
}
