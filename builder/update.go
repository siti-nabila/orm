package builder

import (
	"fmt"
	"sort"
	"strings"

	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
	"github.com/siti-nabila/orm/pkg/helper"
)

func BuildUpdateQuery(
	v any,
	d dialect.Dialector,
	cfg config.Config,
	mode config.PlaceholderMode,
	fields ...map[string]any,
) (UpdateQueryResult, error) {
	if err := helper.IsAllowedPointerStruct(v); err != nil {
		return UpdateQueryResult{}, err
	}

	switch len(fields) {
	case 0:
		return buildUpdateQueryFromStruct(v, d, cfg, mode)
	case 1:
		return buildUpdateQueryFromMap(v, fields[0], d, cfg, mode)
	default:
		return UpdateQueryResult{}, dictionary.ErrDBTooManyArguments
	}
}

func GenerateUpdateSetQuery(
	d dialect.Dialector,
	mode config.PlaceholderMode,
	quote bool,
	cols []mapper.ColumnMeta,
	startIdx int,
) (string, error) {

	if len(cols) == 0 {
		return "", dictionary.ErrDBQueryEmpty
	}

	out := make([]string, len(cols))

	for i, col := range cols {
		colName := col.Name
		if quote {
			colName = d.QuoteIdentifier(colName)
		}

		ph, err := GeneratePlaceholder(d, mode, startIdx+i, col)
		if err != nil {
			return "", err
		}

		out[i] = fmt.Sprintf("%s = %s", colName, ph)
	}

	return strings.Join(out, config.QuerySeperator), nil
}

func GenerateWherePrimaryKeyQuery(
	d dialect.Dialector,
	mode config.PlaceholderMode,
	quote bool,
	col mapper.ColumnMeta,
	idx int,
) (string, error) {
	colName := col.Name
	if quote {
		colName = d.QuoteIdentifier(colName)
	}

	ph, err := GeneratePlaceholder(d, mode, idx, col)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s = %s", colName, ph), nil
}

func buildUpdateQueryFromStruct(
	v any,
	d dialect.Dialector,
	cfg config.Config,
	mode config.PlaceholderMode,
) (UpdateQueryResult, error) {
	// parse struct menjadi meta
	meta, err := mapper.Parse(v, cfg.UseSnakeCase)
	if err != nil {
		return UpdateQueryResult{}, err
	}
	// ambil Primary Key Column
	pk := meta.GetPrimaryKeyColumn()
	if pk == nil {
		return UpdateQueryResult{}, dictionary.ErrPrimaryKeyNotFound
	}

	if helper.IsZero(pk.Value) {
		return UpdateQueryResult{}, dictionary.ErrPrimaryKeyEmpty
	}

	setCols := filterUpdateColumns(meta.Columns)
	if len(setCols) == 0 {
		return UpdateQueryResult{}, dictionary.ErrDBQueryEmpty
	}
	setQuery, err := GenerateUpdateSetQuery(
		d,
		mode,
		cfg.QuoteIdentifier,
		setCols,
		1,
	)
	if err != nil {
		return UpdateQueryResult{}, err
	}
	whereQuery, err := GenerateWherePrimaryKeyQuery(
		d,
		mode,
		cfg.QuoteIdentifier,
		*pk,
		len(setCols)+1,
	)
	if err != nil {
		return UpdateQueryResult{}, err
	}

	args := GenerateValuesFromMeta(setCols)
	args = append(args, pk.Value)

	placeholderCols := make([]mapper.ColumnMeta, 0, len(setCols)+1)
	placeholderCols = append(placeholderCols, setCols...)
	placeholderCols = append(placeholderCols, *pk)

	table := meta.Table
	if cfg.QuoteIdentifier {
		table = d.QuoteIdentifier(table)
	}
	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		table,
		setQuery,
		whereQuery,
	)

	return UpdateQueryResult{
		Query:           query,
		Args:            args,
		PKColumn:        pk,
		FilteredCols:    setCols,
		PlaceholderCols: placeholderCols,
	}, nil
}

func buildUpdateQueryFromMap(
	v any,
	fields map[string]any,
	d dialect.Dialector,
	cfg config.Config,
	mode config.PlaceholderMode,
) (UpdateQueryResult, error) {
	// parse struct menjadi meta
	meta, err := mapper.Parse(v, cfg.UseSnakeCase)
	if err != nil {
		return UpdateQueryResult{}, err
	}

	// ambil Primary Key Column dari struct
	pk := meta.GetPrimaryKeyColumn()
	if pk == nil {
		return UpdateQueryResult{}, dictionary.ErrPrimaryKeyNotFound
	}

	if len(fields) == 0 {
		return UpdateQueryResult{}, dictionary.ErrDBQueryEmpty
	}

	// cek apakah map fields untuk diupdate ada primary key nya yang cocok dengan struct tabler
	pkValue, ok := fields[pk.Name]
	if !ok {
		return UpdateQueryResult{}, dictionary.ErrPrimaryKeyNotFound
	}

	// cek apakah value primary key nya zero value
	if helper.IsZero(pkValue) {
		return UpdateQueryResult{}, dictionary.ErrPrimaryKeyEmpty
	}

	// buat urutan fields map, karena kalau hanya looping map nya saja akan random
	keys := make([]string, 0, len(fields)-1)
	for key := range fields {
		idx, exists := meta.ColumnIndex[key]
		if !exists {
			return UpdateQueryResult{}, fmt.Errorf("%w: %s", dictionary.ErrColumnNotFound, key)
		}

		colMeta := meta.Columns[idx]
		if colMeta.PrimaryKey {
			continue
		}

		keys = append(keys, key)
	}
	sort.Strings(keys)

	if len(keys) == 0 {
		return UpdateQueryResult{}, dictionary.ErrDBQueryEmpty
	}

	setCols := make([]mapper.ColumnMeta, 0, len(keys))
	args := make([]any, 0, len(keys)+1)

	for _, key := range keys {
		idx := meta.ColumnIndex[key]
		colMeta := meta.Columns[idx]
		colMeta.Value = fields[key]

		setCols = append(setCols, colMeta)
		args = append(args, fields[key])
	}

	// generate query set bagian SET col1 = ?, col2 = ? berdasarkan urutan keys yang sudah di sort (ini query set placeholder)
	setQuery, err := GenerateUpdateSetQuery(
		d,
		mode,
		cfg.QuoteIdentifier,
		setCols,
		1,
	)
	if err != nil {
		return UpdateQueryResult{}, err
	}

	whereCol := *pk
	whereCol.Value = pkValue

	placeholderCols := make([]mapper.ColumnMeta, 0, len(setCols)+1)
	placeholderCols = append(placeholderCols, setCols...)
	placeholderCols = append(placeholderCols, whereCol)

	// karena where selalu di akhir, maka index placeholder nya dimulai dari len(setCols)+1
	whereQuery, err := GenerateWherePrimaryKeyQuery(
		d,
		mode,
		cfg.QuoteIdentifier,
		whereCol,
		len(setCols)+1,
	)
	if err != nil {
		return UpdateQueryResult{}, err
	}

	args = append(args, pkValue)

	table := meta.Table
	if cfg.QuoteIdentifier {
		table = d.QuoteIdentifier(table)
	}

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		table,
		setQuery,
		whereQuery,
	)

	return UpdateQueryResult{
		Query:           query,
		Args:            args,
		PKColumn:        pk,
		FilteredCols:    setCols,
		PlaceholderCols: placeholderCols,
	}, nil
}

// fungsi untuk skip kolom primary key pada update query
func filterUpdateColumns(cols []mapper.ColumnMeta) []mapper.ColumnMeta {
	out := make([]mapper.ColumnMeta, 0, len(cols))
	for _, c := range cols {
		if c.PrimaryKey {
			continue
		}
		out = append(out, c)
	}
	return out
}
