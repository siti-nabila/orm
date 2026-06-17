package query_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pagination"
	"github.com/siti-nabila/orm/pkg/dictionary"
	"github.com/siti-nabila/orm/query"
)

const totalItemsSQLFragment = "COUNT(*) OVER() AS __orm_total_items"

type paginationTestUser struct {
	ID   int64  `sql:"column:id;primaryKey"`
	Name string `sql:"column:name"`
}

func (paginationTestUser) TableName() string {
	return "users"
}

type paginationTestORM struct {
	dialect dialect.Dialector
	config  config.Config
	total   *int64
	rows    []paginationTestUser
}

func (o *paginationTestORM) Dialect() dialect.Dialector {
	return o.dialect
}

func (o *paginationTestORM) Config() config.Config {
	return o.config
}

func (o *paginationTestORM) PlaceholderMode() config.PlaceholderMode {
	return config.PlaceholderByNumber
}

func (o *paginationTestORM) ScanQuery(
	_ context.Context,
	_ string,
	_ []any,
	_ []mapper.ColumnMeta,
	dest any,
) error {
	reflect.ValueOf(dest).Elem().Set(reflect.ValueOf(append([]paginationTestUser(nil), o.rows...)))
	return nil
}

func (o *paginationTestORM) ScanPageQuery(
	_ context.Context,
	_ string,
	_ []any,
	_ []mapper.ColumnMeta,
	dest any,
	_ string,
) (*int64, error) {
	reflect.ValueOf(dest).Elem().Set(reflect.ValueOf(append([]paginationTestUser(nil), o.rows...)))
	return o.total, nil
}

func TestDryRunPaginateByDialect(t *testing.T) {
	tests := []struct {
		name       string
		dialect    dialect.Dialector
		wantSuffix string
	}{
		{name: "postgres", dialect: dialect.NewPostgres(), wantSuffix: " LIMIT 21 OFFSET 20"},
		{name: "mysql", dialect: dialect.NewMysql(), wantSuffix: " LIMIT 21 OFFSET 20"},
		{name: "oracle", dialect: dialect.NewOracle(), wantSuffix: " OFFSET 20 ROWS FETCH NEXT 21 ROWS ONLY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &paginationTestORM{dialect: tt.dialect}
			result, err := query.New(o).
				Table(paginationTestUser{}).
				Where("name = ?", "nabila").
				OrderBy("id ASC").
				DryRunPaginate(pagination.Params{Page: 2, Limit: 20})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasSuffix(result.Query, tt.wantSuffix) {
				t.Fatalf("unexpected query: %s", result.Query)
			}
		})
	}
}

func TestMySQLOffsetWithoutLimit(t *testing.T) {
	o := &paginationTestORM{dialect: dialect.NewMysql()}
	result, err := query.New(o).
		Table(paginationTestUser{}).
		Offset(20).
		DryRun()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(result.Query, " LIMIT 18446744073709551615 OFFSET 20") {
		t.Fatalf("unexpected query: %s", result.Query)
	}
}

func TestDryRunCountByDialect(t *testing.T) {
	tests := []struct {
		name           string
		dialect        dialect.Dialector
		whereFragments []string
	}{
		{
			name:    "postgres",
			dialect: dialect.NewPostgres(),
			whereFragments: []string{
				"users.active = $1",
				"users.name = $2",
				"(roles.name = $3 OR roles.id IN ($4, $5))",
			},
		},
		{
			name:    "mysql",
			dialect: dialect.NewMysql(),
			whereFragments: []string{
				"users.active = ?",
				"users.name = ?",
				"(roles.name = ? OR roles.id IN (?, ?))",
			},
		},
		{
			name:    "oracle",
			dialect: dialect.NewOracle(),
			whereFragments: []string{
				"users.active = :1",
				"users.name = :2",
				"(roles.name = :3 OR roles.id IN (:4, :5))",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &paginationTestORM{dialect: tt.dialect}
			result, err := query.New(o).
				Table(paginationTestUser{}).
				Join("roles", "roles.user_id = users.id").
				Where("users.active = ?", true).
				OrWhere("users.name = ?", "nabila").
				WhereGroup(func(q *query.QueryBuilder) {
					q.Where("roles.name = ?", "admin").
						OrWhereIn("roles.id", []int64{1, 2})
				}).
				OrderBy("users.id DESC").
				Limit(20).
				Offset(40).
				DryRunCount()
			if err != nil {
				t.Fatal(err)
			}

			if !strings.HasPrefix(result.Query, "SELECT COUNT(*) FROM users JOIN roles ON roles.user_id = users.id WHERE ") {
				t.Fatalf("unexpected count query: %s", result.Query)
			}
			for _, fragment := range tt.whereFragments {
				if !strings.Contains(result.Query, fragment) {
					t.Fatalf("missing count query fragment %q in %s", fragment, result.Query)
				}
			}
			for _, forbidden := range []string{"ORDER BY", "LIMIT", "OFFSET", "FETCH NEXT"} {
				if strings.Contains(result.Query, forbidden) {
					t.Fatalf("count query should not contain %q: %s", forbidden, result.Query)
				}
			}
			if !reflect.DeepEqual(result.Args, []any{true, "nabila", "admin", int64(1), int64(2)}) {
				t.Fatalf("unexpected count args: %+v", result.Args)
			}
			if result.Mode != builder.DryRunModeQueryRow {
				t.Fatalf("unexpected dry run mode: %s", result.Mode)
			}
		})
	}
}

func TestDryRunCountDoesNotMutateDataQueryAndArgsAreIndependent(t *testing.T) {
	o := &paginationTestORM{dialect: dialect.NewPostgres()}
	q := query.New(o).
		Table(paginationTestUser{}).
		Where("name = ?", "nabila").
		OrderBy("id ASC").
		Limit(10).
		Offset(20)

	countResult, err := q.DryRunCount()
	if err != nil {
		t.Fatal(err)
	}
	if len(countResult.Args) != 1 {
		t.Fatalf("unexpected count args: %+v", countResult.Args)
	}
	countResult.Args[0] = "changed"

	dataResult, err := q.DryRun()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(dataResult.Query, "ORDER BY id ASC") ||
		!strings.HasSuffix(dataResult.Query, " LIMIT 10 OFFSET 20") {
		t.Fatalf("data query lost order or pagination: %s", dataResult.Query)
	}
	if !reflect.DeepEqual(dataResult.Args, []any{"nabila"}) {
		t.Fatalf("data args should be independent from count args: %+v", dataResult.Args)
	}
}

func TestDryRunCountRejectsRawSelectExpr(t *testing.T) {
	o := &paginationTestORM{dialect: dialect.NewPostgres()}
	_, err := query.New(o).
		Table(paginationTestUser{}).
		Select("COUNT(*)").
		DryRunCount()
	if !sameError(err, dictionary.ErrPaginationTotalUnsupported) {
		t.Fatalf("expected ErrPaginationTotalUnsupported, got %v", err)
	}
}

func TestDryRunPaginateUsesORMConfig(t *testing.T) {
	o := &paginationTestORM{
		dialect: dialect.NewPostgres(),
		config: config.Config{
			Pagination: pagination.Config{
				DefaultLimit: 25,
				MaxLimit:     50,
			},
		},
	}

	defaultResult, err := query.New(o).
		Table(paginationTestUser{}).
		DryRunPaginate(pagination.Params{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(defaultResult.Query, " LIMIT 26 OFFSET 0") {
		t.Fatalf("unexpected default query: %s", defaultResult.Query)
	}

	maxResult, err := query.New(o).
		Table(paginationTestUser{}).
		DryRunPaginate(pagination.Params{Limit: pagination.LimitMax})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(maxResult.Query, " LIMIT 51 OFFSET 0") {
		t.Fatalf("unexpected max query: %s", maxResult.Query)
	}
}

func TestPaginateTrimsLookaheadAndDoesNotMutateBuilder(t *testing.T) {
	o := &paginationTestORM{
		dialect: dialect.NewPostgres(),
		rows: []paginationTestUser{
			{ID: 1},
			{ID: 2},
			{ID: 3},
		},
	}
	q := query.New(o).Table(paginationTestUser{}).OrderBy("id ASC")

	var users []paginationTestUser
	meta, err := q.Paginate(&users, pagination.Params{Page: 1, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 || !meta.HasNext {
		t.Fatalf("unexpected result: users=%+v meta=%+v", users, meta)
	}

	dryRun, err := q.DryRun()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(dryRun.Query, "LIMIT") || strings.Contains(dryRun.Query, "OFFSET") {
		t.Fatalf("builder was mutated: %s", dryRun.Query)
	}
}

func TestPaginateWithTotal(t *testing.T) {
	total := int64(42)
	o := &paginationTestORM{
		dialect: dialect.NewPostgres(),
		total:   &total,
		rows:    []paginationTestUser{{ID: 21}, {ID: 22}},
	}
	ordinaryQuery := query.New(o).Table(paginationTestUser{}).WithTotal().Select("COUNT(*)")
	ordinaryDryRun, err := ordinaryQuery.DryRun()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(ordinaryDryRun.Query, "id, name") {
		t.Fatalf("WithTotal changed ordinary query: %s", ordinaryDryRun.Query)
	}

	q := query.New(o).Table(paginationTestUser{}).OrderBy("id ASC").WithTotal()
	dryRun, err := q.DryRunPaginate(pagination.Params{Page: 2, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(dryRun.Query, totalItemsSQLFragment) {
		t.Fatalf("missing window total: %s", dryRun.Query)
	}
	if !strings.HasSuffix(dryRun.Query, " LIMIT 20 OFFSET 20") {
		t.Fatalf("unexpected pagination clause: %s", dryRun.Query)
	}

	var users []paginationTestUser
	meta, err := q.Paginate(&users, pagination.Params{Page: 2, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if meta.TotalItems == nil || *meta.TotalItems != total || meta.TotalPages == nil || *meta.TotalPages != 3 {
		t.Fatalf("unexpected meta: %+v", meta)
	}
}

func TestPaginateWithTotalRejectsJoin(t *testing.T) {
	o := &paginationTestORM{dialect: dialect.NewPostgres()}
	q := query.New(o).
		Table(paginationTestUser{}).
		Join("roles", "roles.user_id = users.id").
		WithTotal()

	_, err := q.DryRunPaginate(pagination.Params{Page: 1, Limit: 20})
	if !sameError(err, dictionary.ErrPaginationTotalWithJoin) {
		t.Fatalf("expected ErrPaginationTotalWithJoin, got %v", err)
	}
}

func TestDryRunPaginateWithoutORMReturnsQueryEmpty(t *testing.T) {
	_, err := query.New(nil).
		Table(paginationTestUser{}).
		DryRunPaginate(pagination.Params{Page: 1, Limit: 20})
	if !sameError(err, dictionary.ErrDBQueryEmpty) {
		t.Fatalf("expected ErrDBQueryEmpty, got %v", err)
	}
}

func TestPaginateWithoutORMReturnsQueryEmpty(t *testing.T) {
	var users []paginationTestUser

	_, err := query.New(nil).
		Table(paginationTestUser{}).
		Paginate(&users, pagination.Params{Page: 1, Limit: 20})
	if !sameError(err, dictionary.ErrDBQueryEmpty) {
		t.Fatalf("expected ErrDBQueryEmpty, got %v", err)
	}
}

func sameError(got, want error) bool {
	if got == nil || want == nil {
		return got == want
	}
	return got.Error() == want.Error()
}
