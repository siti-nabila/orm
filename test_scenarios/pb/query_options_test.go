package pb_test

import (
	"reflect"
	"testing"

	orm "github.com/siti-nabila/orm/orm"
	ormpb "github.com/siti-nabila/orm/pb"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

func TestQueryOptionsFromProto(t *testing.T) {
	got, err := ormpb.QueryOptionsFromProto(&ormpb.QueryOptions{
		Page:       2,
		Limit:      25,
		OffsetMode: ormpb.PaginationOffsetMode_PAGINATION_OFFSET_MODE_IN_MEMORY,
		Select: []string{
			"id",
			"email",
		},
		Sort: []*ormpb.SortField{
			{Field: "createdAt", Desc: true},
		},
		Search: &ormpb.SearchQuery{
			Fields:  []string{"name", "email"},
			Keyword: "nabila",
			Mode:    ormpb.SearchMode_SEARCH_MODE_PREFIX,
		},
		SearchAnd: &ormpb.SearchQueryAnd{
			Fields: []*ormpb.SearchField{
				{Field: "status", Keyword: "ACTIVE"},
			},
		},
		Filters: []*ormpb.Filter{
			{
				Field:    "status",
				Operator: ormpb.FilterOperator_FILTER_OPERATOR_EQUAL,
				Value:    "ACTIVE",
			},
			{
				Field:  "roleCode",
				Values: []string{"10", "20"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	want := orm.QueryOptions{
		Page:       2,
		Limit:      25,
		OffsetMode: orm.OffsetModeInMemory,
		Select:     []string{"id", "email"},
		Sort: []orm.SortField{
			{Field: "createdAt", Desc: true},
		},
		Search: &orm.SearchQuery{
			Fields:  []string{"name", "email"},
			Keyword: "nabila",
			Mode:    orm.SearchModePrefix,
		},
		SearchAnd: &orm.SearchQueryAnd{
			Fields: []*orm.SearchField{
				{Field: "status", Keyword: "ACTIVE"},
			},
		},
		Filters: []orm.Filter{
			{
				Field:    "status",
				Operator: orm.OpEqual,
				Value:    "ACTIVE",
			},
			{
				Field:  "roleCode",
				Values: []string{"10", "20"},
			},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected query options:\ngot=%#v\nwant=%#v", got, want)
	}
}

func TestQueryOptionsFromProtoRejectsInvalidOffsetMode(t *testing.T) {
	_, err := ormpb.QueryOptionsFromProto(&ormpb.QueryOptions{
		OffsetMode: ormpb.PaginationOffsetMode(99),
	})
	if !sameError(err, dictionary.ErrInvalidPaginationOffsetMode) {
		t.Fatalf("expected ErrInvalidPaginationOffsetMode, got %v", err)
	}
}

func TestQueryOptionsFromProtoRejectsInvalidFilterOperator(t *testing.T) {
	_, err := ormpb.QueryOptionsFromProto(&ormpb.QueryOptions{
		Filters: []*ormpb.Filter{
			{Field: "status", Value: "ACTIVE"},
		},
	})
	if !sameError(err, dictionary.ErrInvalidWhereOperator) {
		t.Fatalf("expected ErrInvalidWhereOperator, got %v", err)
	}
}

func TestQueryOptionsFromProtoNilInput(t *testing.T) {
	got, err := ormpb.QueryOptionsFromProto(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, orm.QueryOptions{}) {
		t.Fatalf("unexpected query options: %+v", got)
	}
}

func sameError(got, want error) bool {
	if got == nil || want == nil {
		return got == want
	}
	return got.Error() == want.Error()
}
