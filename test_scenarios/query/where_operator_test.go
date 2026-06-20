package query_test

import (
	"reflect"
	"testing"

	"github.com/siti-nabila/orm/dialect"
	ormapi "github.com/siti-nabila/orm/orm"
	"github.com/siti-nabila/orm/pkg/dictionary"
	"github.com/siti-nabila/orm/query"
)

func TestOperatorString(t *testing.T) {
	if query.OpLessThanEqual.String() != "less_than_equal" {
		t.Fatalf("unexpected operator string: %s", query.OpLessThanEqual.String())
	}
}

func TestOperatorSQL(t *testing.T) {
	tests := []struct {
		name string
		op   query.Operator
		want string
	}{
		{name: "equal", op: query.OpEqual, want: "="},
		{name: "not equal", op: query.OpNotEqual, want: "<>"},
		{name: "less than", op: query.OpLessThan, want: "<"},
		{name: "less than equal", op: query.OpLessThanEqual, want: "<="},
		{name: "greater than", op: query.OpGreaterThan, want: ">"},
		{name: "greater than equal", op: query.OpGreaterThanEqual, want: ">="},
		{name: "like", op: query.OpLike, want: "LIKE"},
		{name: "not like", op: query.OpNotLike, want: "NOT LIKE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.op.SQL()
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("unexpected SQL operator: got %q want %q", got, tt.want)
			}
		})
	}
}

func TestOperatorSQLInvalidOperatorReturnsDictionaryError(t *testing.T) {
	_, err := query.Operator("invalid").SQL()
	if !sameError(err, dictionary.ErrInvalidWhereOperator) {
		t.Fatalf("expected ErrInvalidWhereOperator, got %v", err)
	}
}

func TestORMOperatorAlias(t *testing.T) {
	got, err := ormapi.OpEqual.SQL()
	if err != nil {
		t.Fatal(err)
	}
	if got != "=" {
		t.Fatalf("unexpected SQL operator: %s", got)
	}
}

func TestWhereOpMatchesWhere(t *testing.T) {
	tests := []struct {
		name    string
		dialect dialect.Dialector
	}{
		{name: "postgres", dialect: dialect.NewPostgres()},
		{name: "mysql", dialect: dialect.NewMysql()},
		{name: "oracle", dialect: dialect.NewOracle()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &paginationTestORM{dialect: tt.dialect}

			gotBuilder, err := query.New(o).
				Table(paginationTestUser{}).
				WhereOp("age", query.OpLessThanEqual, 30)
			if err != nil {
				t.Fatal(err)
			}

			got, err := gotBuilder.DryRun()
			if err != nil {
				t.Fatal(err)
			}

			want, err := query.New(o).
				Table(paginationTestUser{}).
				Where("age <= ?", 30).
				DryRun()
			if err != nil {
				t.Fatal(err)
			}

			if got.Query != want.Query {
				t.Fatalf("unexpected query:\ngot:  %s\nwant: %s", got.Query, want.Query)
			}
			if !reflect.DeepEqual(got.Args, want.Args) {
				t.Fatalf("unexpected args: got %+v want %+v", got.Args, want.Args)
			}
		})
	}
}

func TestOrWhereOpMatchesOrWhereAndPreservesArgsOrder(t *testing.T) {
	tests := []struct {
		name    string
		dialect dialect.Dialector
	}{
		{name: "postgres", dialect: dialect.NewPostgres()},
		{name: "mysql", dialect: dialect.NewMysql()},
		{name: "oracle", dialect: dialect.NewOracle()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &paginationTestORM{dialect: tt.dialect}

			gotBuilder, err := query.New(o).
				Table(paginationTestUser{}).
				Where("active = ?", true).
				OrWhereOp("name", query.OpLike, "na%")
			if err != nil {
				t.Fatal(err)
			}

			got, err := gotBuilder.DryRun()
			if err != nil {
				t.Fatal(err)
			}

			want, err := query.New(o).
				Table(paginationTestUser{}).
				Where("active = ?", true).
				OrWhere("name LIKE ?", "na%").
				DryRun()
			if err != nil {
				t.Fatal(err)
			}

			if got.Query != want.Query {
				t.Fatalf("unexpected query:\ngot:  %s\nwant: %s", got.Query, want.Query)
			}
			if !reflect.DeepEqual(got.Args, []any{true, "na%"}) {
				t.Fatalf("unexpected args order: %+v", got.Args)
			}
			if !reflect.DeepEqual(got.Args, want.Args) {
				t.Fatalf("unexpected args: got %+v want %+v", got.Args, want.Args)
			}
		})
	}
}

func TestWhereOpInvalidOperatorReturnsDictionaryError(t *testing.T) {
	o := &paginationTestORM{dialect: dialect.NewPostgres()}

	_, err := query.New(o).
		Table(paginationTestUser{}).
		WhereOp("age", query.Operator("invalid"), 30)
	if !sameError(err, dictionary.ErrInvalidWhereOperator) {
		t.Fatalf("expected ErrInvalidWhereOperator, got %v", err)
	}
}

func TestWhereOrWhereRemainUnchanged(t *testing.T) {
	o := &paginationTestORM{dialect: dialect.NewPostgres()}

	result, err := query.New(o).
		Table(paginationTestUser{}).
		Where("age >= ?", 18).
		OrWhere("name NOT LIKE ?", "test%").
		DryRun()
	if err != nil {
		t.Fatal(err)
	}

	if result.Query != "SELECT id, name FROM users WHERE age >= $1 OR name NOT LIKE $2" {
		t.Fatalf("unexpected query: %s", result.Query)
	}
	if !reflect.DeepEqual(result.Args, []any{18, "test%"}) {
		t.Fatalf("unexpected args: %+v", result.Args)
	}
}
