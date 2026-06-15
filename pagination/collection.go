package pagination

import "sort"

type Collection[T any] struct {
	data       []T
	predicates []func(T) bool
	less       func(a, b T) bool
	config     Config
}

func FromSlice[T any](data []T) *Collection[T] {
	return FromSliceWithConfig(data, Config{})
}

func FromSliceWithConfig[T any](data []T, cfg Config) *Collection[T] {
	return &Collection[T]{
		data:   append([]T(nil), data...),
		config: cfg,
	}
}

func (c *Collection[T]) Filter(predicate func(T) bool) *Collection[T] {
	if predicate != nil {
		c.predicates = append(c.predicates, predicate)
	}
	return c
}

func (c *Collection[T]) Sort(less func(a, b T) bool) *Collection[T] {
	c.less = less
	return c
}

func (c *Collection[T]) Paginate(params Params) (Result[T], error) {
	params, err := NormalizeWithConfig(params, c.config)
	if err != nil {
		return Result[T]{}, err
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
	start := Offset(params)
	if start >= len(filtered) {
		return Result[T]{
			Data: []T{},
			Meta: BuildMeta(params, 0, false, &total),
		}, nil
	}

	end := min(start+params.Limit, len(filtered))
	data := append([]T(nil), filtered[start:end]...)

	return Result[T]{
		Data: data,
		Meta: BuildMeta(params, len(data), end < len(filtered), &total),
	}, nil
}

func (c *Collection[T]) matches(item T) bool {
	for _, predicate := range c.predicates {
		if !predicate(item) {
			return false
		}
	}
	return true
}
