package builder

import (
	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
)

func BuildInsertQueryWithOptions(
	meta *mapper.Meta,
	d dialect.Dialector,
	cfg config.Config,
	mode config.PlaceholderMode,
	opts InsertBuildOptions,
) (InsertAdvancedQueryResult, error) {
	switch d.Type() {
	case dialect.DialectPostgres:
		return buildPostgresInsertAdvancedQuery(meta, d, cfg, mode, opts)
	case dialect.DialectMySQL:
		return buildMySQLInsertAdvancedQuery(meta, d, cfg, mode, opts)
	case dialect.DialectOracle:
		return buildOracleInsertAdvancedQuery(meta, d, cfg, mode, opts)
	default:
		return InsertAdvancedQueryResult{}, nil
	}
}

func quoteConflictName(
	name string,
	d dialect.Dialector,
	quote bool,
) string {
	if quote {
		return d.QuoteIdentifier(name)
	}
	return name
}
