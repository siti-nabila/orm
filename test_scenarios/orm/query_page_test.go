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
	total int64
	rows  []queryPageUser
	calls []queryPageCall
}

func (o *queryPageORM) Dialect() dialect.Dialector {
	return dialect.NewPostgres()
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
