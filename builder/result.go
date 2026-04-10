package builder

import "github.com/siti-nabila/orm/mapper"

const (
	DryRunModeExec     DryRunMode = "exec"
	DryRunModeQuery    DryRunMode = "query"
	DryRunModeQueryRow DryRunMode = "query_row"
)

type (
	DryRunMode        string
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
	DryRunResult struct {
		Query string
		Args  []any
		Mode  DryRunMode
	}
)

func (d DryRunMode) String() string {
	return string(d)
}
