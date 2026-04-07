package builder

import (
	"strings"

	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

func BuildInsertBulkQuery(
	metas []*mapper.Meta,
	table string,
	filteredCols []mapper.ColumnMeta,
	primaryKeyColName string,
	primaryKeyColIndexes []int,
	d dialect.Dialector,
	cfg config.Config,
	mode config.PlaceholderMode,
) (InsertBulkQueryResult, error) {
	var result InsertBulkQueryResult

	if len(metas) == 0 {
		return result, dictionary.ErrBulkInsertEmptyMetas
	}

	if len(filteredCols) == 0 {
		return result, dictionary.ErrDBQueryEmpty
	}

	tableName := table
	if cfg.QuoteIdentifier {
		tableName = d.QuoteIdentifier(tableName)
	}

	colList := GenerateColumnListQuery(d, cfg.QuoteIdentifier, filteredCols)
	if colList == "" {
		return result, dictionary.ErrDBQueryEmpty
	}

	args := make([]any, 0, len(metas)*len(filteredCols))
	valueGroups := make([]string, 0, len(metas))
	placeholderIndex := 1
	effectiveMode := resolveBulkPlaceholderMode(d, mode)

	for _, meta := range metas {
		if meta == nil {
			return result, dictionary.ErrBulkInsertEmptyMetas
		}

		rowFilteredCols, ok := filterInsertColumnsByNames(meta.Columns, filteredCols)
		if !ok || len(rowFilteredCols) != len(filteredCols) {
			return result, dictionary.ErrBulkInsertColumnMismatch
		}

		rowPlaceholderQuery, err := GenerateBulkPlaceholderQuery(
			d,
			effectiveMode,
			rowFilteredCols,
			placeholderIndex,
		)
		if err != nil {
			return result, err
		}

		valueGroups = append(valueGroups, "("+rowPlaceholderQuery+")")
		args = append(args, GenerateValuesFromMeta(rowFilteredCols)...)
		placeholderIndex += len(rowFilteredCols)
	}

	query, err := buildBulkInsertQueryByDialect(
		d,
		tableName,
		colList,
		valueGroups,
		primaryKeyColName,
		cfg.QuoteIdentifier,
	)
	if err != nil {
		return result, err
	}

	result = InsertBulkQueryResult{
		Query:                query,
		Args:                 args,
		PrimaryKeyColName:    primaryKeyColName,
		PrimaryKeyColIndexes: primaryKeyColIndexes,
		FilteredCols:         filteredCols,
		RowCount:             len(metas),
	}

	return result, nil
}

func buildBulkInsertQueryByDialect(
	d dialect.Dialector,
	tableName string,
	colList string,
	valueGroups []string,
	primaryKeyColName string,
	quoteIdentifier bool,
) (string, error) {
	if tableName == "" || colList == "" || len(valueGroups) == 0 {
		return "", dictionary.ErrDBQueryEmpty
	}
	commonQuery := "INSERT INTO " + tableName + "(" + colList + ") VALUES " + strings.Join(valueGroups, ", ")
	switch d.Type() {
	case dialect.DialectOracle:
		return buildOracleInsertAllQuery(tableName, colList, valueGroups), nil

	case dialect.DialectPostgres:
		query := commonQuery
		if primaryKeyColName != "" {
			pkCol := primaryKeyColName
			if quoteIdentifier {
				pkCol = d.QuoteIdentifier(pkCol)
			}
			query += " RETURNING " + pkCol
		}

		return query, nil

	case dialect.DialectMySQL:
		query := commonQuery
		return query, nil

	default:
		return "", dictionary.ErrUnsupportedDialect
	}
}

func buildOracleInsertAllQuery(
	tableName string,
	colList string,
	valueGroups []string,
) string {
	var sb strings.Builder

	sb.WriteString("INSERT ALL ")

	for _, group := range valueGroups {
		sb.WriteString("INTO ")
		sb.WriteString(tableName)
		sb.WriteString("(")
		sb.WriteString(colList)
		sb.WriteString(") VALUES ")
		sb.WriteString(group)
		sb.WriteString(" ")
	}

	sb.WriteString("SELECT 1 FROM dual")

	return sb.String()
}

// filterInsertColumnsByNames menyusun ulang kolom row agar mengikuti urutan baseline.
// Return bool=false kalau ada baseline column yang tidak ditemukan di row.
func filterInsertColumnsByNames(
	cols []mapper.ColumnMeta,
	baseline []mapper.ColumnMeta,
) ([]mapper.ColumnMeta, bool) {
	if len(cols) == 0 || len(baseline) == 0 {
		return nil, false
	}

	colMap := make(map[string]mapper.ColumnMeta, len(cols))
	for _, col := range cols {
		colMap[col.Name] = col
	}

	filtered := make([]mapper.ColumnMeta, 0, len(baseline))
	for _, baseCol := range baseline {
		col, ok := colMap[baseCol.Name]
		if !ok {
			return nil, false
		}
		filtered = append(filtered, col)
	}

	return filtered, true
}

func resolveBulkPlaceholderMode(
	d dialect.Dialector,
	mode config.PlaceholderMode,
) config.PlaceholderMode {
	switch d.Type() {
	case dialect.DialectOracle:
		return config.PlaceholderByNumber
	default:
		return mode
	}
}
