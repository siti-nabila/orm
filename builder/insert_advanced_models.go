package builder

import "github.com/siti-nabila/orm/mapper"

const (
	ConflictAssignInserted ResolvedConflictExprMode = "inserted"
	ConflictAssignValue    ResolvedConflictExprMode = "value"
	ConflictAssignInc      ResolvedConflictExprMode = "inc"
)

type (
	ResolvedConflictExprMode string

	InsertBuildOptions struct {
		ReturningCols []mapper.ColumnMeta
		OnConflict    *OnConflictClause
	}

	OnConflictClause struct {
		TargetCols  []mapper.ColumnMeta
		DoNothing   bool
		Assignments []ResolvedConflictAssignment
	}

	ResolvedConflictAssignment struct {
		ColumnMeta mapper.ColumnMeta
		Mode       ResolvedConflictExprMode
		Value      any
		RefColumn  *mapper.ColumnMeta
	}
)
