package query_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/pagination"
	"github.com/siti-nabila/orm/query"
)

func TestDryRunScanPaginateSimpleCountRemainsSimple(t *testing.T) {
	o := &paginationTestORM{dialect: dialect.NewPostgres()}

	result, err := query.New(o).
		Table(paginationTestUser{}).
		Where("active = ?", true).
		OrderBy("id DESC").
		DryRunScanPaginate(pagination.PaginationOptions{Page: 2, PerPage: 10})
	if err != nil {
		t.Fatal(err)
	}

	if result.Count.Query != "SELECT COUNT(*) FROM users WHERE active = $1" {
		t.Fatalf("simple count should stay simple: %s", result.Count.Query)
	}
	if strings.Contains(result.Count.Query, "count_table") {
		t.Fatalf("simple count should not use wrapper: %s", result.Count.Query)
	}
	if !strings.HasSuffix(result.Data.Query, " ORDER BY id DESC LIMIT 10 OFFSET 10") {
		t.Fatalf("unexpected data query: %s", result.Data.Query)
	}
	if !reflect.DeepEqual(result.Count.Args, []any{true}) ||
		!reflect.DeepEqual(result.Data.Args, []any{true}) {
		t.Fatalf("unexpected args: count=%+v data=%+v", result.Count.Args, result.Data.Args)
	}
}

func TestDryRunScanPaginateComplexCountWrapperByDialect(t *testing.T) {
	tests := []struct {
		name             string
		dialect          dialect.Dialector
		whereFragment    string
		havingFragment   string
		paginationSuffix string
	}{
		{
			name:             "postgres",
			dialect:          dialect.NewPostgres(),
			whereFragment:    "users.active = $1",
			havingFragment:   "COUNT(*) > $2",
			paginationSuffix: " LIMIT 10 OFFSET 20",
		},
		{
			name:             "mysql",
			dialect:          dialect.NewMysql(),
			whereFragment:    "users.active = ?",
			havingFragment:   "COUNT(*) > ?",
			paginationSuffix: " LIMIT 10 OFFSET 20",
		},
		{
			name:             "oracle",
			dialect:          dialect.NewOracle(),
			whereFragment:    "users.active = :1",
			havingFragment:   "COUNT(*) > :2",
			paginationSuffix: " OFFSET 20 ROWS FETCH NEXT 10 ROWS ONLY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &paginationTestORM{dialect: tt.dialect}

			result, err := query.New(o).
				Table(paginationTestUser{}).
				Select("name", "COUNT(*) AS total").
				Join("roles r", "r.user_id = users.id AND r.deleted_at IS NULL").
				Where("users.active = ?", true).
				GroupBy("name").
				Having("COUNT(*) > ?", 2).
				OrderBy("total DESC").
				Limit(99).
				Offset(99).
				DryRunScanPaginate(pagination.PaginationOptions{Page: 3, PerPage: 10})
			if err != nil {
				t.Fatal(err)
			}

			if !strings.HasPrefix(result.Count.Query, "SELECT COUNT(*) FROM (SELECT name, COUNT(*) AS total FROM users JOIN roles r ON r.user_id = users.id AND r.deleted_at IS NULL") ||
				!strings.HasSuffix(result.Count.Query, ") count_table") {
				t.Fatalf("unexpected wrapped count query: %s", result.Count.Query)
			}
			for _, fragment := range []string{
				tt.whereFragment,
				"GROUP BY name",
				"HAVING " + tt.havingFragment,
			} {
				if !strings.Contains(result.Count.Query, fragment) {
					t.Fatalf("missing count fragment %q in %s", fragment, result.Count.Query)
				}
			}
			for _, forbidden := range []string{"ORDER BY", "LIMIT", "OFFSET", "FETCH NEXT"} {
				if strings.Contains(result.Count.Query, forbidden) {
					t.Fatalf("count wrapper should not contain %q: %s", forbidden, result.Count.Query)
				}
			}
			if strings.Contains(result.Count.Query, "AS count_table") {
				t.Fatalf("count wrapper should not use AS alias syntax: %s", result.Count.Query)
			}

			if !strings.Contains(result.Data.Query, "SELECT name, COUNT(*) AS total FROM users JOIN roles r ON") ||
				!strings.Contains(result.Data.Query, "GROUP BY name HAVING "+tt.havingFragment) ||
				!strings.Contains(result.Data.Query, "ORDER BY total DESC") ||
				!strings.HasSuffix(result.Data.Query, tt.paginationSuffix) {
				t.Fatalf("unexpected data query: %s", result.Data.Query)
			}

			wantArgs := []any{true, 2}
			if !reflect.DeepEqual(result.Count.Args, wantArgs) {
				t.Fatalf("unexpected count args: %+v", result.Count.Args)
			}
			if !reflect.DeepEqual(result.Data.Args, wantArgs) {
				t.Fatalf("unexpected data args: %+v", result.Data.Args)
			}
			if result.Count.Mode != builder.DryRunModeQueryRow ||
				result.Data.Mode != builder.DryRunModeQuery {
				t.Fatalf("unexpected dry run modes: count=%s data=%s", result.Count.Mode, result.Data.Mode)
			}
		})
	}
}

func TestDryRunScanPaginateComplexCountWrapperShapes(t *testing.T) {
	tests := []struct {
		name          string
		build         func(*query.QueryBuilder) *query.QueryBuilder
		countFragment string
		dataFragment  string
	}{
		{
			name: "distinct",
			build: func(q *query.QueryBuilder) *query.QueryBuilder {
				return q.Select("name").Distinct()
			},
			countFragment: "SELECT DISTINCT name FROM users",
			dataFragment:  "SELECT DISTINCT name FROM users",
		},
		{
			name: "select expression",
			build: func(q *query.QueryBuilder) *query.QueryBuilder {
				return q.Select("COUNT(*) AS total")
			},
			countFragment: "SELECT COUNT(*) AS total FROM users",
			dataFragment:  "SELECT COUNT(*) AS total FROM users",
		},
		{
			name: "alias select",
			build: func(q *query.QueryBuilder) *query.QueryBuilder {
				return q.Select("users.name AS name")
			},
			countFragment: "SELECT users.name AS name FROM users",
			dataFragment:  "SELECT users.name AS name FROM users",
		},
		{
			name: "group by",
			build: func(q *query.QueryBuilder) *query.QueryBuilder {
				return q.Select("name").GroupBy("name")
			},
			countFragment: "SELECT name FROM users GROUP BY name",
			dataFragment:  "SELECT name FROM users GROUP BY name",
		},
		{
			name: "having",
			build: func(q *query.QueryBuilder) *query.QueryBuilder {
				return q.Select("name", "COUNT(*) AS total").GroupBy("name").Having("COUNT(*) > ?", 1)
			},
			countFragment: "SELECT name, COUNT(*) AS total FROM users GROUP BY name HAVING COUNT(*) > $1",
			dataFragment:  "SELECT name, COUNT(*) AS total FROM users GROUP BY name HAVING COUNT(*) > $1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &paginationTestORM{dialect: dialect.NewPostgres()}

			result, err := tt.build(query.New(o).Table(paginationTestUser{})).
				OrderBy("name ASC").
				DryRunScanPaginate(pagination.PaginationOptions{Page: 1, PerPage: 5})
			if err != nil {
				t.Fatal(err)
			}

			if !strings.HasPrefix(result.Count.Query, "SELECT COUNT(*) FROM (") ||
				!strings.Contains(result.Count.Query, tt.countFragment) ||
				!strings.HasSuffix(result.Count.Query, ") count_table") {
				t.Fatalf("unexpected count wrapper for %s: %s", tt.name, result.Count.Query)
			}
			if strings.Contains(result.Count.Query, "ORDER BY") ||
				strings.Contains(result.Count.Query, "LIMIT") ||
				strings.Contains(result.Count.Query, "OFFSET") {
				t.Fatalf("count wrapper should omit ordering and pagination: %s", result.Count.Query)
			}
			if !strings.Contains(result.Data.Query, tt.dataFragment) ||
				!strings.Contains(result.Data.Query, "ORDER BY name ASC") ||
				!strings.HasSuffix(result.Data.Query, " LIMIT 5 OFFSET 0") {
				t.Fatalf("unexpected data query for %s: %s", tt.name, result.Data.Query)
			}
		})
	}
}

func TestDryRunCountComplexWrapperDoesNotMutateBuilder(t *testing.T) {
	o := &paginationTestORM{dialect: dialect.NewPostgres()}
	q := query.New(o).
		Table(paginationTestUser{}).
		Select("name", "COUNT(*) AS total").
		GroupBy("name").
		Having("COUNT(*) > ?", 1).
		OrderBy("total DESC").
		Limit(10).
		Offset(20)

	countResult, err := q.DryRunCount()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(countResult.Query, "ORDER BY") ||
		strings.Contains(countResult.Query, "LIMIT") ||
		strings.Contains(countResult.Query, "OFFSET") {
		t.Fatalf("count wrapper should omit order and pagination: %s", countResult.Query)
	}

	dataResult, err := q.DryRun()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(dataResult.Query, "ORDER BY total DESC") ||
		!strings.HasSuffix(dataResult.Query, " LIMIT 10 OFFSET 20") {
		t.Fatalf("builder was mutated by DryRunCount: %s", dataResult.Query)
	}
}
