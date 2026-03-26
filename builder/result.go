package builder

import "github.com/siti-nabila/orm/mapper"

type (
	InsertQueryResult struct {
		Query        string
		Args         []any
		PKColumn     *mapper.ColumnMeta
		FilteredCols []mapper.ColumnMeta
	}
	UpdateQueryResult struct {
		Query           string
		Args            []any
		PKColumn        *mapper.ColumnMeta
		FilteredCols    []mapper.ColumnMeta
		PlaceholderCols []mapper.ColumnMeta
	}
)
