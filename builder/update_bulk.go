package builder

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

type UpdateBulkQueryResult struct {
	Query        string
	Args         []any
	FilteredCols []mapper.ColumnMeta
	RowCount     int
}

func BuildUpdateBulkQuery(
	metas []*mapper.Meta,
	table string,
	setCols []mapper.ColumnMeta,
	pkCol mapper.ColumnMeta,
	d dialect.Dialector,
	cfg config.Config,
	mode config.PlaceholderMode,
) (UpdateBulkQueryResult, error) {
	var result UpdateBulkQueryResult

	if len(metas) == 0 {
		return result, dictionary.ErrBulkUpdateEmptyMetas
	}

	if len(setCols) == 0 {
		return result, dictionary.ErrDBQueryEmpty
	}

	if pkCol.Name == "" {
		return result, dictionary.ErrPrimaryKeyNotFound
	}

	effectiveMode := resolveBulkPlaceholderMode(d, mode)

	switch d.Type() {
	case dialect.DialectPostgres:
		return buildBulkUpdatePostgres(metas, table, setCols, pkCol, d, cfg, effectiveMode)
	case dialect.DialectMySQL:
		return buildBulkUpdateMySQL(metas, table, setCols, pkCol, d, cfg, effectiveMode)
	case dialect.DialectOracle:
		return buildBulkUpdateOracle(metas, table, setCols, pkCol, d, cfg, effectiveMode)
	default:
		return result, dictionary.ErrUnsupportedDialect
	}
}

// buildBulkUpdatePostgres generates:
// UPDATE "table" SET "col1" = v.col1, "col2" = v.col2
// FROM (VALUES ($1,$2,$3), ($4,$5,$6)) AS v(pk, col1, col2)
// WHERE "table"."pk" = v.pk
func buildBulkUpdatePostgres(
	metas []*mapper.Meta,
	table string,
	setCols []mapper.ColumnMeta,
	pkCol mapper.ColumnMeta,
	d dialect.Dialector,
	cfg config.Config,
	mode config.PlaceholderMode,
) (UpdateBulkQueryResult, error) {
	var result UpdateBulkQueryResult

	tableName := table
	if cfg.QuoteIdentifier {
		tableName = d.QuoteIdentifier(tableName)
	}

	// allCols = pk + setCols (order for VALUES rows)
	allCols := make([]mapper.ColumnMeta, 0, 1+len(setCols))
	allCols = append(allCols, pkCol)
	allCols = append(allCols, setCols...)

	args := make([]any, 0, len(metas)*len(allCols))
	valueGroups := make([]string, 0, len(metas))
	placeholderIndex := 1

	for i, meta := range metas {
		if meta == nil {
			return result, dictionary.ErrBulkUpdateEmptyMetas
		}

		rowCols, err := extractBulkUpdateRowCols(meta, pkCol, setCols)
		if err != nil {
			return result, err
		}

		// First row: add explicit type casts so PostgreSQL infers correct column types
		if i == 0 {
			rowPh, err := generatePgTypedPlaceholders(d, mode, rowCols, placeholderIndex)
			if err != nil {
				return result, err
			}
			valueGroups = append(valueGroups, "("+rowPh+")")
		} else {
			rowPlaceholders, err := GenerateBulkPlaceholderQuery(d, mode, rowCols, placeholderIndex)
			if err != nil {
				return result, err
			}
			valueGroups = append(valueGroups, "("+rowPlaceholders+")")
		}

		args = append(args, GenerateValuesFromMeta(rowCols)...)
		placeholderIndex += len(rowCols)
	}

	// alias column names for VALUES clause: v(pk, col1, col2)
	aliasNames := make([]string, 0, len(allCols))
	aliasNames = append(aliasNames, pkCol.Name)
	for _, col := range setCols {
		aliasNames = append(aliasNames, col.Name)
	}

	// SET clause: "col1" = v.col1, "col2" = v.col2
	setParts := make([]string, 0, len(setCols))
	for _, col := range setCols {
		colName := col.Name
		if cfg.QuoteIdentifier {
			colName = d.QuoteIdentifier(colName)
		}
		setParts = append(setParts, fmt.Sprintf("%s = v.%s", colName, col.Name))
	}

	// WHERE clause
	pkName := pkCol.Name
	if cfg.QuoteIdentifier {
		pkName = d.QuoteIdentifier(pkCol.Name)
	}

	query := fmt.Sprintf(
		"UPDATE %s SET %s FROM (VALUES %s) AS v(%s) WHERE %s.%s = v.%s",
		tableName,
		strings.Join(setParts, config.QuerySeperator),
		strings.Join(valueGroups, ", "),
		strings.Join(aliasNames, ", "),
		tableName,
		pkName,
		pkCol.Name,
	)

	result = UpdateBulkQueryResult{
		Query:        query,
		Args:         args,
		FilteredCols: setCols,
		RowCount:     len(metas),
	}

	return result, nil
}

// buildBulkUpdateMySQL generates:
// UPDATE `table`
// SET `col1` = CASE `pk` WHEN ? THEN ? WHEN ? THEN ? END,
//
//	`col2` = CASE `pk` WHEN ? THEN ? WHEN ? THEN ? END
//
// WHERE `pk` IN (?, ?)
func buildBulkUpdateMySQL(
	metas []*mapper.Meta,
	table string,
	setCols []mapper.ColumnMeta,
	pkCol mapper.ColumnMeta,
	d dialect.Dialector,
	cfg config.Config,
	mode config.PlaceholderMode,
) (UpdateBulkQueryResult, error) {
	var result UpdateBulkQueryResult

	tableName := table
	if cfg.QuoteIdentifier {
		tableName = d.QuoteIdentifier(tableName)
	}

	pkName := pkCol.Name
	if cfg.QuoteIdentifier {
		pkName = d.QuoteIdentifier(pkCol.Name)
	}

	// Extract row data: each row has pk value + set column values
	type rowData struct {
		pkValue   any
		colValues []any
	}

	rows := make([]rowData, 0, len(metas))
	for _, meta := range metas {
		if meta == nil {
			return result, dictionary.ErrBulkUpdateEmptyMetas
		}

		rowCols, err := extractBulkUpdateRowCols(meta, pkCol, setCols)
		if err != nil {
			return result, err
		}

		rd := rowData{
			pkValue:   rowCols[0].Value,
			colValues: make([]any, len(setCols)),
		}
		for i := range setCols {
			rd.colValues[i] = rowCols[i+1].Value
		}
		rows = append(rows, rd)
	}

	// Build CASE expressions for each SET column
	args := make([]any, 0, len(metas)*(len(setCols)*2+1))
	placeholderIndex := 1
	caseParts := make([]string, 0, len(setCols))

	for colIdx, col := range setCols {
		colName := col.Name
		if cfg.QuoteIdentifier {
			colName = d.QuoteIdentifier(colName)
		}

		var sb strings.Builder
		sb.WriteString(colName)
		sb.WriteString(" = CASE ")
		sb.WriteString(pkName)

		for _, row := range rows {
			phPK, err := GenerateBulkPlaceholder(d, mode, placeholderIndex, pkCol)
			if err != nil {
				return result, err
			}
			placeholderIndex++

			valCol := setCols[colIdx]
			valCol.Value = row.colValues[colIdx]
			phVal, err := GenerateBulkPlaceholder(d, mode, placeholderIndex, valCol)
			if err != nil {
				return result, err
			}
			placeholderIndex++

			sb.WriteString(" WHEN ")
			sb.WriteString(phPK)
			sb.WriteString(" THEN ")
			sb.WriteString(phVal)

			args = append(args, row.pkValue, row.colValues[colIdx])
		}

		sb.WriteString(" END")
		caseParts = append(caseParts, sb.String())
	}

	// WHERE pk IN (?, ?)
	inPlaceholders := make([]string, 0, len(rows))
	for _, row := range rows {
		ph, err := GenerateBulkPlaceholder(d, mode, placeholderIndex, pkCol)
		if err != nil {
			return result, err
		}
		placeholderIndex++
		inPlaceholders = append(inPlaceholders, ph)
		args = append(args, row.pkValue)
	}

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s IN (%s)",
		tableName,
		strings.Join(caseParts, config.QuerySeperator),
		pkName,
		strings.Join(inPlaceholders, ", "),
	)

	result = UpdateBulkQueryResult{
		Query:        query,
		Args:         args,
		FilteredCols: setCols,
		RowCount:     len(metas),
	}

	return result, nil
}

// buildBulkUpdateOracle generates:
// MERGE INTO "table" t
// USING (SELECT :1 AS pk, :2 AS col1 FROM dual UNION ALL SELECT :3, :4 FROM dual) s
// ON (t."pk" = s.pk)
// WHEN MATCHED THEN UPDATE SET t."col1" = s.col1
func buildBulkUpdateOracle(
	metas []*mapper.Meta,
	table string,
	setCols []mapper.ColumnMeta,
	pkCol mapper.ColumnMeta,
	d dialect.Dialector,
	cfg config.Config,
	mode config.PlaceholderMode,
) (UpdateBulkQueryResult, error) {
	var result UpdateBulkQueryResult

	tableName := table
	if cfg.QuoteIdentifier {
		tableName = d.QuoteIdentifier(tableName)
	}

	pkName := pkCol.Name
	if cfg.QuoteIdentifier {
		pkName = d.QuoteIdentifier(pkCol.Name)
	}

	// allCols = pk + setCols
	allCols := make([]mapper.ColumnMeta, 0, 1+len(setCols))
	allCols = append(allCols, pkCol)
	allCols = append(allCols, setCols...)

	args := make([]any, 0, len(metas)*len(allCols))
	selectParts := make([]string, 0, len(metas))
	placeholderIndex := 1

	for i, meta := range metas {
		if meta == nil {
			return result, dictionary.ErrBulkUpdateEmptyMetas
		}

		rowCols, err := extractBulkUpdateRowCols(meta, pkCol, setCols)
		if err != nil {
			return result, err
		}

		var sb strings.Builder
		sb.WriteString("SELECT ")

		for j, col := range rowCols {
			if j > 0 {
				sb.WriteString(", ")
			}

			ph, err := GenerateBulkPlaceholder(d, mode, placeholderIndex, col)
			if err != nil {
				return result, err
			}
			placeholderIndex++

			sb.WriteString(ph)
			// Only add alias on the first row
			if i == 0 {
				sb.WriteString(" AS ")
				sb.WriteString(col.Name)
			}

			args = append(args, col.Value)
		}

		sb.WriteString(" FROM dual")
		selectParts = append(selectParts, sb.String())
	}

	// SET clause: t."col1" = s.col1, t."col2" = s.col2
	updateSetParts := make([]string, 0, len(setCols))
	for _, col := range setCols {
		colName := col.Name
		if cfg.QuoteIdentifier {
			colName = d.QuoteIdentifier(col.Name)
		}
		updateSetParts = append(updateSetParts, fmt.Sprintf("t.%s = s.%s", colName, col.Name))
	}

	query := fmt.Sprintf(
		"MERGE INTO %s t USING (%s) s ON (t.%s = s.%s) WHEN MATCHED THEN UPDATE SET %s",
		tableName,
		strings.Join(selectParts, " UNION ALL "),
		pkName,
		pkCol.Name,
		strings.Join(updateSetParts, config.QuerySeperator),
	)

	result = UpdateBulkQueryResult{
		Query:        query,
		Args:         args,
		FilteredCols: setCols,
		RowCount:     len(metas),
	}

	return result, nil
}

// extractBulkUpdateRowCols extracts [pkCol, ...setCols] from a single meta row,
// returning columns in order with their values populated.
func extractBulkUpdateRowCols(
	meta *mapper.Meta,
	pkCol mapper.ColumnMeta,
	setCols []mapper.ColumnMeta,
) ([]mapper.ColumnMeta, error) {
	// Find PK value from the meta
	metaPK := meta.GetPrimaryKeyColumn()
	if metaPK == nil {
		return nil, dictionary.ErrPrimaryKeyNotFound
	}

	if metaPK.Name != pkCol.Name {
		return nil, dictionary.ErrBulkUpdatePrimaryKeyMismatch
	}

	rowCols := make([]mapper.ColumnMeta, 0, 1+len(setCols))
	rowCols = append(rowCols, *metaPK)

	for _, setCol := range setCols {
		idx, exists := meta.ColumnIndex[setCol.Name]
		if !exists {
			return nil, dictionary.ErrBulkUpdateColumnMismatch
		}
		rowCols = append(rowCols, meta.Columns[idx])
	}

	return rowCols, nil
}

// generatePgTypedPlaceholders generates placeholders with explicit PostgreSQL type casts
// for the first VALUES row, e.g. "$1::bigint, $2::text".
// This tells PostgreSQL the correct column types for the entire VALUES table.
// Type is determined from the struct field type (FieldSrc) for accuracy.
func generatePgTypedPlaceholders(
	d dialect.Dialector,
	mode config.PlaceholderMode,
	cols []mapper.ColumnMeta,
	startIndex int,
) (string, error) {
	if len(cols) == 0 {
		return "", nil
	}

	out := make([]string, len(cols))
	for i, col := range cols {
		ph, err := GenerateBulkPlaceholder(d, mode, startIndex+i, col)
		if err != nil {
			return "", err
		}

		pgType := pgTypeFromField(col.FieldSrc)
		if pgType != "" {
			ph += "::" + pgType
		}

		out[i] = ph
	}

	return strings.Join(out, config.QuerySeperator), nil
}

// pgTypeFromField returns the PostgreSQL type name based on the struct field's reflect type.
func pgTypeFromField(field reflect.Value) string {
	if !field.IsValid() {
		return ""
	}

	t := field.Type()

	// Unwrap pointer
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Check time.Time first (it's a struct)
	if t == reflect.TypeOf(time.Time{}) {
		return "timestamptz"
	}

	switch t.Kind() {
	case reflect.Int8, reflect.Int16, reflect.Uint8, reflect.Uint16:
		return "smallint"
	case reflect.Int32, reflect.Uint32:
		return "integer"
	case reflect.Int, reflect.Int64, reflect.Uint, reflect.Uint64:
		return "bigint"
	case reflect.Float32:
		return "real"
	case reflect.Float64:
		return "double precision"
	case reflect.Bool:
		return "boolean"
	case reflect.String:
		return "text"
	default:
		// []byte → bytea
		if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 {
			return "bytea"
		}
		return ""
	}
}
