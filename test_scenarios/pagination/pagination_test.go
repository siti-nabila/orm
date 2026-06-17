package pagination_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/siti-nabila/orm/pagination"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

func TestNormalize(t *testing.T) {
	params, err := pagination.Normalize(pagination.Params{})
	if err != nil {
		t.Fatal(err)
	}
	if params.Page != 1 || params.Limit != pagination.DefaultLimit {
		t.Fatalf("unexpected normalized params: %+v", params)
	}

	params, err = pagination.Normalize(pagination.Params{Limit: pagination.LimitMax})
	if err != nil || params.Limit != pagination.MaxLimit {
		t.Fatalf("unexpected max limit params=%+v err=%v", params, err)
	}

	params, err = pagination.Normalize(pagination.Params{Limit: pagination.MaxLimit + 1})
	if err != nil || params.Limit != pagination.MaxLimit {
		t.Fatalf("unexpected clamped params=%+v err=%v", params, err)
	}

	_, err = pagination.Normalize(pagination.Params{Limit: -2})
	if !sameError(err, dictionary.ErrPaginationInvalidLimit) {
		t.Fatalf("expected ErrPaginationInvalidLimit, got %v", err)
	}

	_, err = pagination.Normalize(pagination.Params{Page: math.MaxInt, Limit: 2})
	if !sameError(err, dictionary.ErrPaginationOffsetOverflow) {
		t.Fatalf("expected ErrPaginationOffsetOverflow, got %v", err)
	}
}

func TestNormalizeWithConfig(t *testing.T) {
	cfg := pagination.Config{DefaultLimit: 25, MaxLimit: 50}

	params, err := pagination.NormalizeWithConfig(pagination.Params{}, cfg)
	if err != nil || params.Limit != 25 {
		t.Fatalf("unexpected default params=%+v err=%v", params, err)
	}

	params, err = pagination.NormalizeWithConfig(pagination.Params{Limit: pagination.LimitMax}, cfg)
	if err != nil || params.Limit != 50 {
		t.Fatalf("unexpected max params=%+v err=%v", params, err)
	}

	params, err = pagination.NormalizeWithConfig(pagination.Params{Limit: 100}, cfg)
	if err != nil || params.Limit != 50 {
		t.Fatalf("unexpected clamped params=%+v err=%v", params, err)
	}
}

func TestCollectionPaginate(t *testing.T) {
	type item struct {
		ID    int
		Group string
	}

	input := []item{
		{ID: 4, Group: "a"},
		{ID: 1, Group: "b"},
		{ID: 3, Group: "a"},
		{ID: 2, Group: "a"},
	}
	original := append([]item(nil), input...)

	result, err := pagination.FromSlice(input).
		Filter(func(v item) bool { return v.Group == "a" }).
		Sort(func(a, b item) bool { return a.ID < b.ID }).
		Paginate(pagination.Params{Page: 1, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}

	want := []item{{ID: 2, Group: "a"}, {ID: 3, Group: "a"}}
	if !reflect.DeepEqual(result.Data, want) {
		t.Fatalf("unexpected page: got=%+v want=%+v", result.Data, want)
	}
	if result.Meta.TotalItems == nil || *result.Meta.TotalItems != 3 {
		t.Fatalf("unexpected total: %+v", result.Meta.TotalItems)
	}
	if !result.Meta.HasNext || result.Meta.HasPrev {
		t.Fatalf("unexpected meta: %+v", result.Meta)
	}
	if !reflect.DeepEqual(input, original) {
		t.Fatalf("input mutated: got=%+v want=%+v", input, original)
	}
}

func TestCollectionOutOfRange(t *testing.T) {
	result, err := pagination.FromSlice([]int{1, 2}).Paginate(pagination.Params{Page: 3, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Data) != 0 {
		t.Fatalf("expected empty data, got %+v", result.Data)
	}
	if result.Meta.TotalItems == nil || *result.Meta.TotalItems != 2 {
		t.Fatalf("unexpected meta: %+v", result.Meta)
	}
}

func TestCollectionWithConfig(t *testing.T) {
	result, err := pagination.FromSliceWithConfig(
		[]int{1, 2, 3, 4},
		pagination.Config{DefaultLimit: 2, MaxLimit: 3},
	).Paginate(pagination.Params{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Data) != 2 || result.Meta.Limit != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func sameError(got, want error) bool {
	if got == nil || want == nil {
		return got == want
	}
	return got.Error() == want.Error()
}
