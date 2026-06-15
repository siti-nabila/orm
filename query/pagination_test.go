package query

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pagination"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

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
			result, err := New(o).
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
	result, err := New(o).
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

	defaultResult, err := New(o).
		Table(paginationTestUser{}).
		DryRunPaginate(pagination.Params{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(defaultResult.Query, " LIMIT 26 OFFSET 0") {
		t.Fatalf("unexpected default query: %s", defaultResult.Query)
	}

	maxResult, err := New(o).
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
	query := New(o).Table(paginationTestUser{}).OrderBy("id ASC")

	var users []paginationTestUser
	meta, err := query.Paginate(&users, pagination.Params{Page: 1, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 || !meta.HasNext {
		t.Fatalf("unexpected result: users=%+v meta=%+v", users, meta)
	}

	dryRun, err := query.DryRun()
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
	ordinaryQuery := New(o).Table(paginationTestUser{}).WithTotal().Select("COUNT(*)")
	ordinaryDryRun, err := ordinaryQuery.DryRun()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(ordinaryDryRun.Query, "id, name") {
		t.Fatalf("WithTotal changed ordinary query: %s", ordinaryDryRun.Query)
	}

	query := New(o).Table(paginationTestUser{}).OrderBy("id ASC").WithTotal()
	dryRun, err := query.DryRunPaginate(pagination.Params{Page: 2, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(dryRun.Query, "COUNT(*) OVER() AS "+totalItemsColumn) {
		t.Fatalf("missing window total: %s", dryRun.Query)
	}
	if !strings.HasSuffix(dryRun.Query, " LIMIT 20 OFFSET 20") {
		t.Fatalf("unexpected pagination clause: %s", dryRun.Query)
	}

	var users []paginationTestUser
	meta, err := query.Paginate(&users, pagination.Params{Page: 2, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if meta.TotalItems == nil || *meta.TotalItems != total || meta.TotalPages == nil || *meta.TotalPages != 3 {
		t.Fatalf("unexpected meta: %+v", meta)
	}
}

func TestPaginateWithTotalRejectsJoin(t *testing.T) {
	o := &paginationTestORM{dialect: dialect.NewPostgres()}
	query := New(o).
		Table(paginationTestUser{}).
		Join("roles", "roles.user_id = users.id").
		WithTotal()

	_, err := query.DryRunPaginate(pagination.Params{Page: 1, Limit: 20})
	if !reflect.DeepEqual(err, dictionary.ErrPaginationTotalWithJoin) {
		t.Fatalf("expected ErrPaginationTotalWithJoin, got %v", err)
	}
}
