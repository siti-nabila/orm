package query

import (
	"strings"

	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

type subqueryCondition struct {
	operator ClauseOperator
	prefix   string
	child    *QueryBuilder
}

func (subqueryCondition) isCondition() {}

// WhereExists snapshots child and renders it with the parent configuration.
func (b *QueryBuilder) WhereExists(child *QueryBuilder) *QueryBuilder {
	return b.whereSubquery("EXISTS", child)
}

func (b *QueryBuilder) WhereNotExists(child *QueryBuilder) *QueryBuilder {
	return b.whereSubquery("NOT EXISTS", child)
}

func (b *QueryBuilder) WhereInQuery(column string, child *QueryBuilder) *QueryBuilder {
	prefix := ""
	if strings.TrimSpace(column) != "" {
		prefix = column + " IN"
	}
	return b.whereSubquery(prefix, child)
}

func (b *QueryBuilder) whereSubquery(prefix string, child *QueryBuilder) *QueryBuilder {
	b.conditions = append(b.conditions, subqueryCondition{ClauseAnd, prefix, child.clone()})
	return b
}

// LeftJoinQuery snapshots a derived SELECT. Alias must be a simple identifier.
// ON expressions follow the same trusted-expression contract as LeftJoin.
func (b *QueryBuilder) LeftJoinQuery(child *QueryBuilder, alias, on string) *QueryBuilder {
	b.joins = append(b.joins, builderJoin{Join: Join{Type: "LEFT JOIN", On: on}, child: child.clone(), alias: alias})
	return b
}

func validSubqueryAlias(alias string) bool {
	if alias == "" {
		return false
	}
	for i, c := range alias {
		if c != '_' && !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') && !(i > 0 && c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

func (b *QueryBuilder) buildJoins(cfg config.Config, d dialect.Dialector, mode config.PlaceholderMode, start int) (string, []any, int, error) {
	var sql strings.Builder
	var args []any
	idx := start
	for _, j := range b.joins {
		table := j.Table
		if table == "" {
			if !validSubqueryAlias(j.alias) || strings.TrimSpace(j.On) == "" {
				return "", nil, start, dictionary.ErrDBQueryEmpty
			}
			res, err := j.child.buildAt(cfg, d, mode, idx)
			if err != nil {
				return "", nil, start, err
			}
			alias := j.alias
			if cfg.QuoteIdentifier {
				alias = d.QuoteIdentifier(alias)
			}
			table = "(" + res.Query + ") " + alias
			args = append(args, res.Args...)
			idx += len(res.Args)
		}
		sql.WriteString(" " + j.Type + " " + table + " ON " + j.On)
	}
	return sql.String(), args, idx, nil
}
