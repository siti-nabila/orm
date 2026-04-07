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
	InsertBulkQueryResult struct {
		Query                string
		Args                 []any
		PrimaryKeyColName    string
		PrimaryKeyColIndexes []int
		FilteredCols         []mapper.ColumnMeta
		RowCount             int
	}
)
