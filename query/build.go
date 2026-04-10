package query

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
	"github.com/siti-nabila/orm/pkg/helper"
)

type (
	QueryBuilderResult struct {
		Query        string
		Args         []any
		SelectedCols []mapper.ColumnMeta
		Mode         builder.DryRunMode
	}
)

func rebindQueryPlaceholders(
	d dialect.Dialector,
	mode config.PlaceholderMode,
	query string,
	startIdx int,
) (string, int, int, error) {
	if query == "" {
		return "", startIdx, 0, nil
	}

	var out strings.Builder
	idx := startIdx
	count := 0

	for _, ch := range query {
		if ch == '?' {
			switch mode {
			case config.PlaceholderByNumber:
				out.WriteString(d.PlaceholderByNumber(idx))
			case config.PlaceholderByName:
				// untuk raw condition string, pakai numbered placeholder dulu
				out.WriteString(d.PlaceholderByNumber(idx))
			default:
				return "", startIdx, 0, dictionary.ErrDBPlaceholder
			}
			idx++
			count++
			continue
		}

		out.WriteRune(ch)
	}

	return out.String(), idx, count, nil
}

func buildExprCondition(
	d dialect.Dialector,
	mode config.PlaceholderMode,
	c ExpressionCondition,
	startIdx int,
) (string, []any, int, error) {
	expandedQuery, expandedArgs, err := expandSliceArgsInQuery(c.Query, c.Args)
	if err != nil {
		return "", nil, startIdx, err
	}

	rebound, nextIdx, placeholderCount, err := rebindQueryPlaceholders(d, mode, expandedQuery, startIdx)
	if err != nil {
		return "", nil, startIdx, err
	}

	if placeholderCount != len(expandedArgs) {
		return "", nil, startIdx, fmt.Errorf(
			"placeholder count does not match args count: query=%q placeholders=%d args=%d",
			expandedQuery,
			placeholderCount,
			len(expandedArgs),
		)
	}

	return rebound, expandedArgs, nextIdx, nil
}
func buildGroupCondition(
	d dialect.Dialector,
	mode config.PlaceholderMode,
	c GroupCondition,
	startIdx int,
) (string, []any, int, error) {
	groupQuery, groupArgs, nextIdx, err := buildConditions(d, mode, c.Conditions, startIdx)
	if err != nil {
		return "", nil, startIdx, err
	}

	if groupQuery == "" {
		return "", nil, startIdx, nil
	}

	return "(" + groupQuery + ")", groupArgs, nextIdx, nil
}

func appendConditionPart(parts []string, idx int, op ClauseOperator, expr string) []string {
	if expr == "" {
		return parts
	}

	if idx == 0 {
		return append(parts, expr)
	}

	return append(parts, string(op)+" "+expr)
}

func buildConditions(
	d dialect.Dialector,
	mode config.PlaceholderMode,
	conds []Condition,
	startIdx int,
) (string, []any, int, error) {
	if len(conds) == 0 {
		return "", nil, startIdx, nil
	}

	parts := make([]string, 0, len(conds))
	args := make([]any, 0)
	idx := startIdx
	partIndex := 0

	for _, cond := range conds {
		switch c := cond.(type) {
		case ExpressionCondition:
			expr, exprArgs, nextIdx, err := buildExprCondition(d, mode, c, idx)
			if err != nil {
				return "", nil, startIdx, err
			}

			if expr != "" {
				parts = appendConditionPart(parts, partIndex, c.Operator, expr)
				args = append(args, exprArgs...)
				idx = nextIdx
				partIndex++
			}

		case GroupCondition:
			expr, exprArgs, nextIdx, err := buildGroupCondition(d, mode, c, idx)
			if err != nil {
				return "", nil, startIdx, err
			}

			if expr != "" {
				parts = appendConditionPart(parts, partIndex, c.Operator, expr)
				args = append(args, exprArgs...)
				idx = nextIdx
				partIndex++
			}

		default:
			return "", nil, startIdx, fmt.Errorf("unsupported condition type")
		}
	}

	return strings.Join(parts, " "), args, idx, nil
}

func resolveSelectedColumns(all []mapper.ColumnMeta, picked []string) ([]mapper.ColumnMeta, error) {
	if len(picked) == 0 {
		out := make([]mapper.ColumnMeta, 0, len(all))
		out = append(out, all...)
		return out, nil
	}

	index := make(map[string]mapper.ColumnMeta, len(all))
	for _, col := range all {
		index[col.Name] = col
	}

	out := make([]mapper.ColumnMeta, 0, len(picked))
	for _, name := range picked {
		col, ok := index[name]
		if !ok {
			return nil, fmt.Errorf("selected column not found: %s", name)
		}
		out = append(out, col)
	}

	return out, nil
}

func buildSelectColumnList(d dialect.Dialector, quote bool, cols []mapper.ColumnMeta) string {
	out := make([]string, 0, len(cols))
	for _, col := range cols {
		name := col.Name
		if quote {
			name = d.QuoteIdentifier(name)
		}
		out = append(out, name)
	}
	return strings.Join(out, ", ")
}

func expandSliceArgsInQuery(query string, args []any) (string, []any, error) {
	if query == "" {
		return "", nil, nil
	}

	var (
		out      strings.Builder
		flatArgs = make([]any, 0)
		argIndex = 0
	)

	for _, ch := range query {
		if ch != '?' {
			out.WriteRune(ch)
			continue
		}

		if argIndex >= len(args) {
			return "", nil, fmt.Errorf("placeholder count exceeds args count")
		}

		arg := args[argIndex]
		argIndex++

		if helper.IsExpandableSliceArg(arg) {
			rv := reflect.ValueOf(arg)

			// empty slice handling
			if rv.Len() == 0 {
				out.WriteString("NULL")
				continue
			}

			for i := 0; i < rv.Len(); i++ {
				if i > 0 {
					out.WriteString(", ")
				}
				out.WriteString("?")
				flatArgs = append(flatArgs, rv.Index(i).Interface())
			}
			continue
		}

		out.WriteString("?")
		flatArgs = append(flatArgs, arg)
	}

	if argIndex != len(args) {
		return "", nil, fmt.Errorf("args count exceeds placeholder count")
	}

	return out.String(), flatArgs, nil
}

func (b *QueryBuilder) build() (QueryBuilderResult, error) {
	if b.orm == nil {
		return QueryBuilderResult{}, dictionary.ErrDBQueryEmpty
	}

	if b.model == nil {
		return QueryBuilderResult{}, dictionary.ErrDBQueryEmpty
	}

	cfg := b.orm.Config()
	d := b.orm.Dialect()
	mode := b.orm.PlaceholderMode()

	meta, err := mapper.Parse(b.model, cfg.UseSnakeCase)
	if err != nil {
		return QueryBuilderResult{}, err
	}

	table := meta.Table
	if cfg.QuoteIdentifier {
		table = d.QuoteIdentifier(table)
	}

	var (
		selectParts  []string
		selectedCols []mapper.ColumnMeta
	)

	if len(b.selectCols) > 0 {
		selectedCols, err = resolveSelectedColumns(meta.Columns, b.selectCols)
		if err != nil {
			return QueryBuilderResult{}, err
		}

		if len(selectedCols) > 0 {
			selectParts = append(selectParts, buildSelectColumnParts(d, cfg.QuoteIdentifier, selectedCols)...)
		}
	} else if len(b.selectExprs) == 0 {
		selectedCols = append(selectedCols, meta.Columns...)
		selectParts = append(selectParts, buildSelectColumnParts(d, cfg.QuoteIdentifier, meta.Columns)...)
	}

	if len(b.selectExprs) > 0 {
		selectParts = append(selectParts, b.selectExprs...)
	}

	if len(selectParts) == 0 {
		return QueryBuilderResult{}, dictionary.ErrDBQueryEmpty
	}

	selectQuery := strings.Join(selectParts, ", ")
	query := fmt.Sprintf("SELECT %s FROM %s", selectQuery, table)

	for _, j := range b.joins {
		query += " " + j.Type + " " + j.Table + " ON " + j.On
	}

	whereQuery, args, _, err := buildConditions(d, mode, b.conditions, 1)
	if err != nil {
		return QueryBuilderResult{}, err
	}

	if whereQuery != "" {
		query += " WHERE " + whereQuery
	}

	if len(b.orderBys) > 0 {
		query += " ORDER BY " + strings.Join(b.orderBys, ", ")
	}

	if b.limit != nil {
		query += " LIMIT " + fmt.Sprint(*b.limit)
	}

	if b.offset != nil {
		query += " OFFSET " + fmt.Sprint(*b.offset)
	}

	dryRunMode := builder.DryRunModeQuery
	if b.singleRow {
		dryRunMode = builder.DryRunModeQueryRow
	}

	return QueryBuilderResult{
		Query:        query,
		Args:         args,
		SelectedCols: selectedCols,
		Mode:         dryRunMode,
	}, nil
}

func (b *QueryBuilder) Scan(dest any) error {
	if dest == nil {
		return dictionary.ErrDBQueryEmpty
	}

	res, err := b.build()
	if err != nil {
		return err
	}

	return b.orm.ScanQuery(b.ctx, res.Query, res.Args, res.SelectedCols, dest)
}

func buildSelectColumnParts(d dialect.Dialector, quote bool, cols []mapper.ColumnMeta) []string {
	out := make([]string, 0, len(cols))
	for _, col := range cols {
		name := col.Name
		if quote {
			name = d.QuoteIdentifier(name)
		}
		out = append(out, name)
	}
	return out
}
