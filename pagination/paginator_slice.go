package pagination

import "sort"

type SlicePaginator[T any] struct {
	data       []T
	predicates []func(T) bool
	less       func(a, b T) bool
	config     Config
}

func FromSlice[T any](data []T) *SlicePaginator[T] {
	return FromSliceWithConfig(data, Config{})
}

func FromSliceWithConfig[T any](data []T, cfg Config) *SlicePaginator[T] {
	return &SlicePaginator[T]{
		data:   append([]T(nil), data...),
		config: cfg,
	}
}

func (c *SlicePaginator[T]) Filter(predicate func(T) bool) *SlicePaginator[T] {
	if predicate != nil {
		c.predicates = append(c.predicates, predicate)
	}
	return c
}

func (c *SlicePaginator[T]) Sort(less func(a, b T) bool) *SlicePaginator[T] {
	c.less = less
	return c
}

func (c *SlicePaginator[T]) Paginate(params Params) (PageData[T], error) {
	params, err := NormalizeWithConfig(params, c.config)
	if err != nil {
		return EmptyPageData[T](params.Page, params.Limit), err
	}

	filtered := make([]T, 0, len(c.data))
	for _, item := range c.data {
		if c.matches(item) {
			filtered = append(filtered, item)
		}
	}

	if c.less != nil {
		sort.SliceStable(filtered, func(i, j int) bool {
			return c.less(filtered[i], filtered[j])
		})
	}

	total := int64(len(filtered))
	meta := BuildPageMeta(PaginationOptions{
		Page:    params.Page,
		PerPage: params.Limit,
	}, total)

	start := Offset(params)
	if start >= len(filtered) {
		return NewPageData([]T{}, meta), nil
	}

	end := min(start+params.Limit, len(filtered))
	data := append([]T(nil), filtered[start:end]...)

	return NewPageData(data, meta), nil
}

func (c *SlicePaginator[T]) matches(item T) bool {
	for _, predicate := range c.predicates {
		if !predicate(item) {
			return false
		}
	}
	return true
}
