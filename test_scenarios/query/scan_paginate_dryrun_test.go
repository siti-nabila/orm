package query_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/orm"
	"github.com/siti-nabila/orm/pagination"
	"github.com/siti-nabila/orm/pkg/dictionary"
	"github.com/siti-nabila/orm/query"
)

type profileSearchRow struct {
	AuthID    int64  `sql:"column:auth_id"`
	Email     string `sql:"column:email"`
	ProfileID int64  `sql:"column:profile_id"`
	Name      string `sql:"column:name"`
	Address   string `sql:"column:address"`
	Phone     string `sql:"column:phone"`
}

func (profileSearchRow) TableName() string {
	return "profile p"
}

func TestDryRunScanPaginateSimpleQueryByDialect(t *testing.T) {
	tests := []struct {
		name       string
		dialect    dialect.Dialector
		wantSuffix string
	}{
		{name: "postgres", dialect: dialect.NewPostgres(), wantSuffix: " LIMIT 20 OFFSET 0"},
		{name: "mysql", dialect: dialect.NewMysql(), wantSuffix: " LIMIT 20 OFFSET 0"},
		{name: "oracle", dialect: dialect.NewOracle(), wantSuffix: " OFFSET 0 ROWS FETCH NEXT 20 ROWS ONLY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &paginationTestORM{dialect: tt.dialect}
			result, err := query.New(o).
				Table(paginationTestUser{}).
				DryRunScanPaginate(pagination.PaginationOptions{Page: 1, PerPage: 20})
			if err != nil {
				t.Fatal(err)
			}

			if result.Count.Query != "SELECT COUNT(*) FROM users" {
				t.Fatalf("unexpected count query: %s", result.Count.Query)
			}
			if len(result.Count.Args) != 0 {
				t.Fatalf("unexpected count args: %+v", result.Count.Args)
			}
			if result.Count.Mode != builder.DryRunModeQueryRow {
				t.Fatalf("unexpected count mode: %s", result.Count.Mode)
			}

			if !strings.HasPrefix(result.Data.Query, "SELECT id, name FROM users") ||
				!strings.HasSuffix(result.Data.Query, tt.wantSuffix) {
				t.Fatalf("unexpected data query: %s", result.Data.Query)
			}
			if len(result.Data.Args) != 0 {
				t.Fatalf("unexpected data args: %+v", result.Data.Args)
			}
			if result.Data.Mode != builder.DryRunModeQuery {
				t.Fatalf("unexpected data mode: %s", result.Data.Mode)
			}
		})
	}
}

func TestDryRunScanPaginateFiltersJoinOrderAndPlaceholdersByDialect(t *testing.T) {
	tests := []struct {
		name              string
		dialect           dialect.Dialector
		whereFragments    []string
		paginationSuffix  string
		invalidAliasToken string
	}{
		{
			name:    "postgres",
			dialect: dialect.NewPostgres(),
			whereFragments: []string{
				"users.active = $1",
				"users.id >= $2",
				"roles.name = $3",
			},
			paginationSuffix: " LIMIT 15 OFFSET 15",
		},
		{
			name:    "mysql",
			dialect: dialect.NewMysql(),
			whereFragments: []string{
				"users.active = ?",
				"users.id >= ?",
				"roles.name = ?",
			},
			paginationSuffix: " LIMIT 15 OFFSET 15",
		},
		{
			name:    "oracle",
			dialect: dialect.NewOracle(),
			whereFragments: []string{
				"users.active = :1",
				"users.id >= :2",
				"roles.name = :3",
			},
			paginationSuffix:  " OFFSET 15 ROWS FETCH NEXT 15 ROWS ONLY",
			invalidAliasToken: "AS count_table",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &paginationTestORM{dialect: tt.dialect}
			q := query.New(o).
				Table(paginationTestUser{}).
				Join("roles", "roles.user_id = users.id").
				Where("users.active = ?", true).
				OrderBy("users.id DESC")

			q, err := q.WhereOp("users.id", query.OpGreaterThanEqual, int64(10))
			if err != nil {
				t.Fatal(err)
			}
			q = q.Where("roles.name = ?", "admin")

			result, err := q.DryRunScanPaginate(pagination.PaginationOptions{
				Page:    2,
				PerPage: 15,
			})
			if err != nil {
				t.Fatal(err)
			}

			if !strings.HasPrefix(result.Count.Query, "SELECT COUNT(*) FROM users JOIN roles ON roles.user_id = users.id WHERE ") {
				t.Fatalf("unexpected count query: %s", result.Count.Query)
			}
			for _, fragment := range tt.whereFragments {
				if !strings.Contains(result.Count.Query, fragment) {
					t.Fatalf("missing count fragment %q in %s", fragment, result.Count.Query)
				}
				if !strings.Contains(result.Data.Query, fragment) {
					t.Fatalf("missing data fragment %q in %s", fragment, result.Data.Query)
				}
			}
			for _, forbidden := range []string{"ORDER BY", "LIMIT", "OFFSET", "FETCH NEXT"} {
				if strings.Contains(result.Count.Query, forbidden) {
					t.Fatalf("count query should not contain %q: %s", forbidden, result.Count.Query)
				}
			}
			if tt.invalidAliasToken != "" && strings.Contains(result.Count.Query, tt.invalidAliasToken) {
				t.Fatalf("count query should avoid invalid alias syntax: %s", result.Count.Query)
			}
			if !strings.Contains(result.Data.Query, "ORDER BY users.id DESC") ||
				!strings.HasSuffix(result.Data.Query, tt.paginationSuffix) {
				t.Fatalf("unexpected data query: %s", result.Data.Query)
			}

			wantArgs := []any{true, int64(10), "admin"}
			if !reflect.DeepEqual(result.Count.Args, wantArgs) {
				t.Fatalf("unexpected count args: %+v", result.Count.Args)
			}
			if !reflect.DeepEqual(result.Data.Args, wantArgs) {
				t.Fatalf("unexpected data args: %+v", result.Data.Args)
			}
		})
	}
}

func TestDryRunScanPaginatePostgresFullTextProfileSearch(t *testing.T) {
	o := &paginationTestORM{dialect: dialect.NewPostgres()}

	q, err := query.New(o).
		Table(profileSearchRow{}).
		Select(
			"a.id AS auth_id",
			"a.email",
			"p.id AS profile_id",
			"p.\"name\"",
			"p.address",
			"p.phone",
		).
		Join("auth a", "a.id = p.user_id").
		Join("user_profile_search ups", "ups.profile_id = p.id").
		WhereFullText("ups.fts_keyword", "nabila@example.com")
	if err != nil {
		t.Fatal(err)
	}

	result, err := q.
		OrderBy("p.id DESC").
		DryRunScanPaginate(pagination.PaginationOptions{
			Page:    2,
			PerPage: 10,
		})
	if err != nil {
		t.Fatal(err)
	}

	for _, fragment := range []string{
		"JOIN auth a ON a.id = p.user_id",
		"JOIN user_profile_search ups ON ups.profile_id = p.id",
		"ups.fts_keyword @@ websearch_to_tsquery('simple', $1)",
	} {
		if !strings.Contains(result.Count.Query, fragment) {
			t.Fatalf("missing count fragment %q in %s", fragment, result.Count.Query)
		}
		if !strings.Contains(result.Data.Query, fragment) {
			t.Fatalf("missing data fragment %q in %s", fragment, result.Data.Query)
		}
	}
	if !strings.HasPrefix(result.Count.Query, "SELECT COUNT(*) FROM (SELECT ") ||
		!strings.HasSuffix(result.Count.Query, ") count_table") {
		t.Fatalf("unexpected count query shape: %s", result.Count.Query)
	}
	for _, forbidden := range []string{"ORDER BY", "LIMIT", "OFFSET"} {
		if strings.Contains(result.Count.Query, forbidden) {
			t.Fatalf("count query should not contain %q: %s", forbidden, result.Count.Query)
		}
	}
	if !strings.Contains(result.Data.Query, "ORDER BY p.id DESC") ||
		!strings.HasSuffix(result.Data.Query, " LIMIT 10 OFFSET 10") {
		t.Fatalf("unexpected data query: %s", result.Data.Query)
	}

	wantArgs := []any{"nabila@example.com"}
	if !reflect.DeepEqual(result.Count.Args, wantArgs) ||
		!reflect.DeepEqual(result.Data.Args, wantArgs) {
		t.Fatalf("unexpected args: count=%+v data=%+v", result.Count.Args, result.Data.Args)
	}
	if result.Count.Mode != builder.DryRunModeQueryRow ||
		result.Data.Mode != builder.DryRunModeQuery {
		t.Fatalf("unexpected dry run modes: count=%s data=%s", result.Count.Mode, result.Data.Mode)
	}
}

func TestWhereFullTextRejectsUnsupportedDialects(t *testing.T) {
	tests := []struct {
		name    string
		dialect dialect.Dialector
	}{
		{name: "mysql", dialect: dialect.NewMysql()},
		{name: "oracle", dialect: dialect.NewOracle()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &paginationTestORM{dialect: tt.dialect}
			_, err := query.New(o).
				Table(profileSearchRow{}).
				WhereFullText("ups.fts_keyword", "nabila")
			if !sameError(err, dictionary.ErrUnsupportedSearchModeForDialect) {
				t.Fatalf("expected ErrUnsupportedSearchModeForDialect, got %v", err)
			}
		})
	}
}

func TestDryRunScanPaginateOptionsValidation(t *testing.T) {
	o := &paginationTestORM{
		dialect: dialect.NewPostgres(),
		config: config.Config{
			Pagination: pagination.Config{
				DefaultLimit: 25,
				MaxLimit:     50,
			},
		},
	}

	invalidPage, err := query.New(o).
		Table(paginationTestUser{}).
		DryRunScanPaginate(pagination.PaginationOptions{Page: -5, PerPage: 10})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(invalidPage.Data.Query, " LIMIT 10 OFFSET 0") {
		t.Fatalf("invalid page should normalize to first page: %s", invalidPage.Data.Query)
	}

	_, err = query.New(o).
		Table(paginationTestUser{}).
		DryRunScanPaginate(pagination.PaginationOptions{Page: 1, PerPage: -1})
	if !sameError(err, dictionary.ErrPaginationInvalidLimit) {
		t.Fatalf("expected ErrPaginationInvalidLimit, got %v", err)
	}

	maxLimit, err := query.New(o).
		Table(paginationTestUser{}).
		DryRunScanPaginate(pagination.PaginationOptions{Page: 2, PerPage: 100, MaxLimit: 30})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(maxLimit.Data.Query, " LIMIT 30 OFFSET 30") {
		t.Fatalf("perPage should be capped by MaxLimit: %s", maxLimit.Data.Query)
	}
}

func TestDryRunScanPaginateORMAliases(t *testing.T) {
	db := orm.NewSqlQueryAdapter(nil, nil, dialect.NewPostgres(), config.Config{})

	result, err := db.UseModel(paginationTestUser{}).
		Where("active = ?", true).
		DryRunScanPaginate(orm.PaginationOptions{Page: 1, PerPage: 5})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Count.Query, "active = $1") ||
		!strings.HasSuffix(result.Data.Query, " LIMIT 5 OFFSET 0") {
		t.Fatalf("unexpected dry run result: %+v", result)
	}
}
