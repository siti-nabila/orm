package pagination

import (
	"math"
	"reflect"
	"testing"

	"github.com/siti-nabila/orm/pkg/dictionary"
)

func TestNormalize(t *testing.T) {
	params, err := Normalize(Params{})
	if err != nil {
		t.Fatal(err)
	}
	if params.Page != 1 || params.Limit != DefaultLimit {
		t.Fatalf("unexpected normalized params: %+v", params)
	}

	params, err = Normalize(Params{Limit: LimitMax})
	if err != nil || params.Limit != MaxLimit {
		t.Fatalf("unexpected max limit params=%+v err=%v", params, err)
	}

	params, err = Normalize(Params{Limit: MaxLimit + 1})
	if err != nil || params.Limit != MaxLimit {
		t.Fatalf("unexpected clamped params=%+v err=%v", params, err)
	}

	_, err = Normalize(Params{Limit: -2})
	if !reflect.DeepEqual(err, dictionary.ErrPaginationInvalidLimit) {
		t.Fatalf("expected ErrPaginationInvalidLimit, got %v", err)
	}

	_, err = Normalize(Params{Page: math.MaxInt, Limit: 2})
	if !reflect.DeepEqual(err, dictionary.ErrPaginationOffsetOverflow) {
		t.Fatalf("expected ErrPaginationOffsetOverflow, got %v", err)
	}
}

func TestNormalizeWithConfig(t *testing.T) {
	cfg := Config{DefaultLimit: 25, MaxLimit: 50}

	params, err := NormalizeWithConfig(Params{}, cfg)
	if err != nil || params.Limit != 25 {
		t.Fatalf("unexpected default params=%+v err=%v", params, err)
	}

	params, err = NormalizeWithConfig(Params{Limit: LimitMax}, cfg)
	if err != nil || params.Limit != 50 {
		t.Fatalf("unexpected max params=%+v err=%v", params, err)
	}

	params, err = NormalizeWithConfig(Params{Limit: 100}, cfg)
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

	result, err := FromSlice(input).
		Filter(func(v item) bool { return v.Group == "a" }).
		Sort(func(a, b item) bool { return a.ID < b.ID }).
		Paginate(Params{Page: 1, Limit: 2})
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
	result, err := FromSlice([]int{1, 2}).Paginate(Params{Page: 3, Limit: 2})
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
	result, err := FromSliceWithConfig(
		[]int{1, 2, 3, 4},
		Config{DefaultLimit: 2, MaxLimit: 3},
	).Paginate(Params{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Data) != 2 || result.Meta.Limit != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
}
