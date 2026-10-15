package query_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/pagination"
	"github.com/siti-nabila/orm/query"
)

func TestComposedQueriesUseParentDialectAndOrderedArguments(t *testing.T) {
	for _, d := range []dialect.Dialector{dialect.NewPostgres(), dialect.NewMysql(), dialect.NewOracle()} {
		t.Run(d.Name(), func(t *testing.T) {
			parent := &paginationTestORM{dialect: d}
			// Deliberately different child dialect/configuration.
			childORM := &paginationTestORM{dialect: dialect.NewMysql(), config: config.Config{QuoteIdentifier: true}}
			child := query.New(childORM).Table(paginationTestUser{}).Select("id", "MIN(name) AS first_name").Where("name = ?", "join").GroupBy("id")
			exists := query.New(childORM).Table(paginationTestUser{}).Select("id").Where("id = users.id").Where("name = ?", "exists")
			q := query.New(parent).Table(paginationTestUser{}).LeftJoinQuery(child, "names", "names.id = users.id").WhereGroup(func(g *query.QueryBuilder) { g.WhereExists(exists).OrWhere("name = ?", "fallback") }).WhereInQuery("id", exists).WhereNotExists(exists).Where("name = ?", "parent").OrderBy("id ASC")
			child.Where("name = ?", "later")
			exists.Where("name = ?", "later")
			before, err := q.DryRun()
			if err != nil {
				t.Fatal(err)
			}
			want := []any{"join", "exists", "fallback", "exists", "exists", "parent"}
			if !reflect.DeepEqual(before.Args, want) || before.Mode != builder.DryRunModeQuery {
				t.Fatalf("%+v", before)
			}
			for i := range want {
				if !strings.Contains(before.Query, d.PlaceholderByNumber(i+1)) {
					t.Fatal(before.Query)
				}
			}
			for _, part := range []string{"LEFT JOIN (SELECT id, MIN(name) AS first_name FROM users", ") names ON", "GROUP BY id", "EXISTS (SELECT id FROM users WHERE id = users.id", "id IN (SELECT", "NOT EXISTS (SELECT"} {
				if !strings.Contains(before.Query, part) {
					t.Fatalf("missing %s in %s", part, before.Query)
				}
			}
			page, err := q.DryRunScanPaginate(pagination.PaginationOptions{Page: 2, PerPage: 3})
			if err != nil || !reflect.DeepEqual(page.Count.Args, want) || !reflect.DeepEqual(page.Data.Args, want) || strings.Contains(page.Count.Query, "ORDER BY") {
				t.Fatalf("%+v %v", page, err)
			}
			after, err := q.DryRun()
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatal("count/page mutated query", after, err)
			}
			own, err := child.DryRun()
			if err != nil || len(own.Args) != 2 || !strings.Contains(own.Query, "`users`") {
				t.Fatal(own, err)
			}
		})
	}
}

func TestSubqueryUpdateOffsetsAndInvalidChildrenFailBeforeExecution(t *testing.T) {
	for _, d := range []dialect.Dialector{dialect.NewPostgres(), dialect.NewMysql(), dialect.NewOracle()} {
		t.Run(d.Name(), func(t *testing.T) {
			f := &fakeORM{dialect: d, mode: config.PlaceholderByNumber}
			child := query.New(f).Table(paginationTestUser{}).Select("id").Where("name = ?", "child")
			q := query.New(f).Table(&whereOnlyUpdateModel{Flag: 0}).WhereInQuery("id", child).WhereExists(child)
			res, err := q.DryRunUpdates()
			if err != nil || res.Mode != builder.DryRunModeExec || !reflect.DeepEqual(res.Args, []any{0, "child", "child"}) || !strings.Contains(res.Query, "name = "+d.PlaceholderByNumber(2)) || !strings.Contains(res.Query, "name = "+d.PlaceholderByNumber(3)) {
				t.Fatal(res, err)
			}
			if _, err = q.Updates(); err != nil || f.calls != 1 {
				t.Fatal(err, f.calls)
			}
			for _, bad := range []*query.QueryBuilder{nil, query.New(f), query.New(f).Table(paginationTestUser{}).Where("name = ?")} {
				q := query.New(f).Table(&whereOnlyUpdateModel{}).WhereExists(bad)
				if _, err := q.DryRun(); err == nil {
					t.Fatal("invalid child accepted")
				}
				if _, err := q.DryRunCount(); err == nil {
					t.Fatal("invalid count child accepted")
				}
				if _, err := q.Updates(); err == nil || f.calls != 1 {
					t.Fatal("invalid child executed", err, f.calls)
				}
			}
		})
	}
}

func TestDerivedJoinRejectsInvalidAliasAndQuotesWithParent(t *testing.T) {
	for _, d := range []dialect.Dialector{dialect.NewPostgres(), dialect.NewMysql(), dialect.NewOracle()} {
		o := &paginationTestORM{dialect: d, config: config.Config{QuoteIdentifier: true}}
		child := query.New(o).Table(paginationTestUser{}).Select("id").Where("name = ?", "child")
		for _, alias := range []string{"", "bad alias", "1bad", "x; DROP TABLE users"} {
			if _, err := query.New(o).Table(paginationTestUser{}).LeftJoinQuery(child, alias, "x.id = users.id").DryRun(); err == nil {
				t.Fatalf("accepted alias %q", alias)
			}
		}
		if _, err := query.New(o).Table(paginationTestUser{}).LeftJoinQuery(nil, "x", "x.id = users.id").DryRun(); err == nil {
			t.Fatal("nil derived child accepted")
		}
		res, err := query.New(o).Table(paginationTestUser{}).LeftJoinQuery(child, "x", "x.id = users.id").DryRun()
		if err != nil || !strings.Contains(res.Query, ") "+d.QuoteIdentifier("x")+" ON") || strings.Contains(res.Query, ") AS ") {
			t.Fatal(res, err)
		}
	}
}

func TestNamedUpdateComposesSubqueryWithoutPlaceholderCollision(t *testing.T) {
	f := &fakeORM{dialect: dialect.NewOracle(), mode: config.PlaceholderByName}
	child := query.New(f).Table(paginationTestUser{}).Select("id").Where("name = ?", "child")
	got, err := query.New(f).Table(&whereOnlyUpdateModel{Flag: 0}).WhereInQuery("id", child).DryRunUpdates()
	if err != nil || !strings.Contains(got.Query, "SET flag = :flag") || !strings.Contains(got.Query, "name = :2") || !reflect.DeepEqual(got.Args, []any{0, "child"}) {
		t.Fatal(got, err)
	}
}

func TestSubquerySnapshotPreservesBoundSlicesAndJoinLiteralCompatibility(t *testing.T) {
	_ = query.Join{Type: "LEFT JOIN", Table: "users", On: "users.id = parent.id"}
	if reflect.TypeOf(query.Join{}).NumField() != 3 {
		t.Fatal("public Join shape changed")
	}
	o := &paginationTestORM{dialect: dialect.NewPostgres()}
	ids := []int{1, 2}
	child := query.New(o).Table(paginationTestUser{}).Select("id").WhereIn("id", ids)
	parent := query.New(o).Table(paginationTestUser{}).WhereExists(child)
	ids[0] = 9
	res, err := parent.DryRun()
	if err != nil || !reflect.DeepEqual(res.Args, []any{1, 2}) {
		t.Fatal(res, err)
	}
}
