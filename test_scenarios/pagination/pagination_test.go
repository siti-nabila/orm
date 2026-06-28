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

func TestBuildPageMeta(t *testing.T) {
	tests := []struct {
		name       string
		opts       pagination.PaginationOptions
		cfg        pagination.Config
		totalRows  int64
		wantMeta   pagination.PageMeta
		wantOffset int
	}{
		{
			name: "total rows zero normalizes page and per page",
			opts: pagination.PaginationOptions{
				Page:    0,
				PerPage: 0,
			},
			totalRows: 0,
			wantMeta: pagination.PageMeta{
				Page:  1,
				Limit: pagination.DefaultLimit,
			},
			wantOffset: 0,
		},
		{
			name: "page greater than one has no prev when total rows zero",
			opts: pagination.PaginationOptions{
				Page:    3,
				PerPage: 10,
			},
			totalRows: 0,
			wantMeta: pagination.PageMeta{
				Page:  3,
				Limit: 10,
			},
			wantOffset: 20,
		},
		{
			name: "exact division calculates total pages",
			opts: pagination.PaginationOptions{
				Page:    1,
				PerPage: 5,
			},
			totalRows: 20,
			wantMeta: pagination.PageMeta{
				Page:       1,
				Limit:      5,
				Total:      20,
				TotalPages: 4,
				HasNext:    true,
			},
			wantOffset: 0,
		},
		{
			name: "remainder rounds total pages up",
			opts: pagination.PaginationOptions{
				Page:    2,
				PerPage: 5,
			},
			totalRows: 23,
			wantMeta: pagination.PageMeta{
				Page:       2,
				Limit:      5,
				Total:      23,
				TotalPages: 5,
				HasNext:    true,
				HasPrev:    true,
			},
			wantOffset: 5,
		},
		{
			name: "last page has previous but no next",
			opts: pagination.PaginationOptions{
				Page:    5,
				PerPage: 5,
			},
			totalRows: 23,
			wantMeta: pagination.PageMeta{
				Page:       5,
				Limit:      5,
				Total:      23,
				TotalPages: 5,
				HasPrev:    true,
			},
			wantOffset: 20,
		},
		{
			name: "page greater than total pages has previous but no next",
			opts: pagination.PaginationOptions{
				Page:    7,
				PerPage: 5,
			},
			totalRows: 23,
			wantMeta: pagination.PageMeta{
				Page:       7,
				Limit:      5,
				Total:      23,
				TotalPages: 5,
				HasPrev:    true,
			},
			wantOffset: 30,
		},
		{
			name: "max limit caps per page before page meta",
			opts: pagination.PaginationOptions{
				Page:     2,
				PerPage:  50,
				MaxLimit: 20,
			},
			totalRows: 45,
			wantMeta: pagination.PageMeta{
				Page:       2,
				Limit:      20,
				Total:      45,
				TotalPages: 3,
				HasNext:    true,
				HasPrev:    true,
			},
			wantOffset: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := pagination.NormalizeOptionsWithConfig(tt.opts, tt.cfg)
			if err != nil {
				t.Fatal(err)
			}

			got := pagination.BuildPageMeta(opts, tt.totalRows)
			if !reflect.DeepEqual(got, tt.wantMeta) {
				t.Fatalf("unexpected page meta: got=%+v want=%+v", got, tt.wantMeta)
			}

			if gotOffset := pagination.OptionsOffset(opts); gotOffset != tt.wantOffset {
				t.Fatalf("unexpected offset: got=%d want=%d", gotOffset, tt.wantOffset)
			}
		})
	}
}

func TestNormalizeOptionsInMemoryOffset(t *testing.T) {
	defaultMode, err := pagination.NormalizeOptionsWithConfig(pagination.PaginationOptions{
		Page:    1,
		PerPage: 10,
	}, pagination.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if defaultMode.InMemoryOffset != nil {
		t.Fatalf("unexpected in-memory offset options: %+v", defaultMode.InMemoryOffset)
	}

	inMemory, err := pagination.NormalizeOptionsWithConfig(pagination.PaginationOptions{
		Page:    1,
		PerPage: 10,
		InMemoryOffset: &pagination.InMemoryOffsetOptions{
			CursorField: "id",
			MaxLimit:    25,
		},
	}, pagination.Config{MaxLimit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if inMemory.InMemoryOffset == nil ||
		inMemory.InMemoryOffset.CursorField != "id" ||
		inMemory.InMemoryOffset.MaxLimit != 25 {
		t.Fatalf("unexpected in-memory offset options: %+v", inMemory.InMemoryOffset)
	}

	capped, err := pagination.NormalizeOptionsWithConfig(pagination.PaginationOptions{
		Page:    1,
		PerPage: 10,
		InMemoryOffset: &pagination.InMemoryOffsetOptions{
			MaxLimit: 100,
		},
	}, pagination.Config{MaxLimit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if capped.InMemoryOffset == nil || capped.InMemoryOffset.MaxLimit != 50 {
		t.Fatalf("unexpected capped in-memory offset options: %+v", capped.InMemoryOffset)
	}
}

func TestSlicePaginatorPaginate(t *testing.T) {
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
	if !reflect.DeepEqual(result.Items, want) {
		t.Fatalf("unexpected page: got=%+v want=%+v", result.Items, want)
	}
	if result.Total != 3 || result.Page != 1 || result.Limit != 2 || result.TotalPages != 2 {
		t.Fatalf("unexpected page data: %+v", result)
	}
	if !result.HasNext || result.HasPrev {
		t.Fatalf("unexpected page data: %+v", result)
	}
	if !reflect.DeepEqual(input, original) {
		t.Fatalf("input mutated: got=%+v want=%+v", input, original)
	}
}

func TestSlicePaginatorOutOfRange(t *testing.T) {
	result, err := pagination.FromSlice([]int{1, 2}).Paginate(pagination.Params{Page: 3, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected empty items, got %+v", result.Items)
	}
	if result.Items == nil {
		t.Fatal("expected empty items slice, got nil")
	}
	if result.Total != 2 || result.TotalPages != 1 || result.HasNext || !result.HasPrev {
		t.Fatalf("unexpected page data: %+v", result)
	}
}

func TestSlicePaginatorWithConfig(t *testing.T) {
	result, err := pagination.FromSliceWithConfig(
		[]int{1, 2, 3, 4},
		pagination.Config{DefaultLimit: 2, MaxLimit: 3},
	).Paginate(pagination.Params{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 2 || result.Limit != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestSlicePaginatorNilInputReturnsEmptyItems(t *testing.T) {
	result, err := pagination.FromSlice[int](nil).Paginate(pagination.Params{Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if result.Items == nil || len(result.Items) != 0 {
		t.Fatalf("expected empty items slice, got %+v", result.Items)
	}
	if result.Total != 0 || result.TotalPages != 0 || result.HasNext || result.HasPrev {
		t.Fatalf("unexpected empty result: %+v", result)
	}
}

func TestSlicePaginatorLastPageHasPreviousAndNoNext(t *testing.T) {
	result, err := pagination.FromSlice([]int{1, 2, 3}).Paginate(pagination.Params{Page: 2, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.Items, []int{3}) {
		t.Fatalf("unexpected items: %+v", result.Items)
	}
	if result.Total != 3 || result.TotalPages != 2 || result.HasNext || !result.HasPrev {
		t.Fatalf("unexpected page data: %+v", result)
	}
}

func TestPageDataHelpers(t *testing.T) {
	meta := pagination.PageMeta{
		Page:       2,
		Limit:      20,
		Total:      55,
		TotalPages: 3,
		HasNext:    true,
		HasPrev:    true,
	}

	data := pagination.NewPageData[int](nil, meta)
	if data.Items == nil || len(data.Items) != 0 {
		t.Fatalf("expected empty items slice, got %+v", data.Items)
	}
	if data.Total != 55 || data.Page != 2 || data.Limit != 20 || data.TotalPages != 3 ||
		!data.HasNext || !data.HasPrev {
		t.Fatalf("unexpected page data: %+v", data)
	}

	empty := pagination.EmptyPageData[int](0, 20)
	if empty.Items == nil || len(empty.Items) != 0 || empty.Page != 1 || empty.Limit != 20 ||
		empty.Total != 0 || empty.TotalPages != 0 || empty.HasNext || empty.HasPrev {
		t.Fatalf("unexpected empty page data: %+v", empty)
	}
}

func sameError(got, want error) bool {
	if got == nil || want == nil {
		return got == want
	}
	return got.Error() == want.Error()
}
