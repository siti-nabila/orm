package builder

import (
	"strconv"
	"strings"

	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

func GeneratePlaceholderQuery(
	d dialect.Dialector,
	mode config.PlaceholderMode,
	cols []mapper.ColumnMeta,
) (string, error) {
	if len(cols) == 0 {
		return "", nil
	}

	out := make([]string, len(cols))
	for idx, col := range cols {
		ph, err := GeneratePlaceholder(d, mode, idx+1, col)
		if err != nil {
			return "", err
		}
		out[idx] = ph
	}

	return strings.Join(out, config.QuerySeperator), nil
}
func GeneratePlaceholder(
	d dialect.Dialector,
	mode config.PlaceholderMode,
	idx int,
	col mapper.ColumnMeta,
) (string, error) {
	switch mode {
	case config.PlaceholderByNumber:
		return d.PlaceholderByNumber(idx), nil
	case config.PlaceholderByName:
		return d.PlaceholderByName(col.Name), nil
	default:
		return "", dictionary.ErrDBPlaceholder
	}
}

func GenerateBulkPlaceholderQuery(
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
		out[i] = ph
	}

	return strings.Join(out, config.QuerySeperator), nil
}

func GenerateBulkPlaceholder(
	d dialect.Dialector,
	mode config.PlaceholderMode,
	idx int,
	col mapper.ColumnMeta,
) (string, error) {
	switch mode {
	case config.PlaceholderByNumber:
		return d.PlaceholderByNumber(idx), nil
	case config.PlaceholderByName:
		return d.PlaceholderByName(col.Name + "_" + strconv.Itoa(idx)), nil
	default:
		return "", dictionary.ErrDBPlaceholder
	}
}
