package query

import (
	"context"

	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
)

type (
	ormQuery interface {
		Dialect() dialect.Dialector
		Config() config.Config
		PlaceholderMode() config.PlaceholderMode
		ScanQuery(ctx context.Context, query string, args []any, selectedCols []mapper.ColumnMeta, dest any) error
	}

	QueryBuilder struct {
		ctx        context.Context
		orm        ormQuery
		model      any
		selectCols []string
		conditions []Condition
		limit      *int
		offset     *int
		orderBys   []string
	}
)

func New(o ormQuery) *QueryBuilder {
	return &QueryBuilder{
		orm: o,
		ctx: context.Background(),
	}
}

func (b *QueryBuilder) Table(v any) *QueryBuilder {
	b.model = v
	return b
}

func (b *QueryBuilder) Select(cols ...string) *QueryBuilder {
	if len(cols) == 0 {
		return b
	}
	b.selectCols = append(b.selectCols, cols...)
	return b
}

func (b *QueryBuilder) Where(query string, args ...any) *QueryBuilder {
	b.conditions = append(b.conditions, ExpressionCondition{
		Operator: ClauseAnd,
		Query:    query,
		Args:     args,
	})

	return b
}

func (b *QueryBuilder) OrWhere(query string, args ...any) *QueryBuilder {
	b.conditions = append(b.conditions, ExpressionCondition{
		Operator: ClauseOr,
		Query:    query,
		Args:     args,
	})

	return b
}

func (b *QueryBuilder) WhereGroup(fn func(q *QueryBuilder)) *QueryBuilder {
	sub := New(nil)
	fn(sub)
	if len(sub.conditions) == 0 {
		return b
	}

	b.conditions = append(b.conditions, GroupCondition{
		Operator:   ClauseAnd,
		Conditions: sub.conditions,
	})
	return b
}

func (b *QueryBuilder) First(dest any) error {
	limit := 1
	b.limit = &limit
	return b.Scan(dest)
}

func (b *QueryBuilder) OrWhereGroup(fn func(q *QueryBuilder)) *QueryBuilder {
	sub := New(nil)
	fn(sub)
	if len(sub.conditions) == 0 {
		return b
	}

	b.conditions = append(b.conditions, GroupCondition{
		Operator:   ClauseOr,
		Conditions: sub.conditions,
	})
	return b
}

func (b *QueryBuilder) Limit(limit int) *QueryBuilder {
	if limit <= 0 {
		return b
	}
	b.limit = &limit
	return b

}

func (b *QueryBuilder) OrderBy(expressions ...string) *QueryBuilder {
	if len(expressions) == 0 {
		return b
	}

	for _, e := range expressions {
		if e == "" {
			continue
		}
		b.orderBys = append(b.orderBys, e)
	}

	return b
}
func (b *QueryBuilder) Offset(n int) *QueryBuilder {
	if n < 0 {
		return b
	}

	b.offset = &n
	return b
}

func (b *QueryBuilder) WithContext(ctx context.Context) *QueryBuilder {
	if ctx == nil {
		return b
	}
	b.ctx = ctx
	return b
}
