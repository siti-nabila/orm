package dryrun_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/orm"
	"github.com/siti-nabila/orm/pagination"
	"github.com/siti-nabila/orm/pkg/dictionary"
	"github.com/siti-nabila/orm/test_scenarios/models"
)

func TestDryRunPaginateWithTotalUsesPublicORMFlow(t *testing.T) {
	tests := []struct {
		name             string
		dialect          dialect.Dialector
		whereFragment    string
		paginationSuffix string
	}{
		{
			name:             "postgres",
			dialect:          dialect.NewPostgres(),
			whereFragment:    "active = $1",
			paginationSuffix: " LIMIT 20 OFFSET 20",
		},
		{
			name:             "mysql",
			dialect:          dialect.NewMysql(),
			whereFragment:    "active = ?",
			paginationSuffix: " LIMIT 20 OFFSET 20",
		},
		{
			name:             "oracle",
			dialect:          dialect.NewOracle(),
			whereFragment:    "active = :1",
			paginationSuffix: " OFFSET 20 ROWS FETCH NEXT 20 ROWS ONLY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := orm.NewSqlQueryAdapter(context.Background(), nil, tt.dialect, config.Config{})

			result, err := db.UseModel(models.PaginationUser{}).
				Where("active = ?", true).
				OrderBy("id ASC").
				WithTotal().
				DryRunPaginate(pagination.Params{Page: 2, Limit: 20})
			if err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(result.Query, "COUNT(*) OVER() AS __orm_total_items") {
				t.Fatalf("missing total window expression: %s", result.Query)
			}
			if !strings.Contains(result.Query, tt.whereFragment) {
				t.Fatalf("missing where fragment %q in query: %s", tt.whereFragment, result.Query)
			}
			if !strings.HasSuffix(result.Query, tt.paginationSuffix) {
				t.Fatalf("unexpected pagination suffix: %s", result.Query)
			}
			if !reflect.DeepEqual(result.Args, []any{true}) {
				t.Fatalf("unexpected args: %+v", result.Args)
			}
			if result.Mode != builder.DryRunModeQuery {
				t.Fatalf("unexpected dry run mode: %s", result.Mode)
			}
		})
	}
}

func TestDryRunPaginateWithTotalRejectsJoinThroughPublicORMFlow(t *testing.T) {
	db := orm.NewSqlQueryAdapter(context.Background(), nil, dialect.NewPostgres(), config.Config{})

	_, err := db.UseModel(models.PaginationUser{}).
		Join("roles", "roles.user_id = users.id").
		WithTotal().
		DryRunPaginate(pagination.Params{Page: 1, Limit: 20})
	if !sameError(err, dictionary.ErrPaginationTotalWithJoin) {
		t.Fatalf("expected ErrPaginationTotalWithJoin, got %v", err)
	}
}

func sameError(got, want error) bool {
	if got == nil || want == nil {
		return got == want
	}
	return got.Error() == want.Error()
}
