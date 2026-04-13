package orm

import (
	"context"

	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/mapper"
)

const (
	advInsScanModeNone advInsScanMode = iota
	advInsScanModeModel
	advInsScanModeExplicit
)

type (
	advInsScanMode uint8
	CreateCommand  struct {
		ctx          context.Context
		orm          *ORM
		v            any
		opts         CreateOptions
		scanIntoDest []any
	}
	CreateOptions struct {
		Returning  []string
		OnConflict *OnConflict
	}
	OnConflict struct {
		TargetColumns []string
		DoNothing     bool
		DoUpdates     []string
		Assignments   []ConflictAssignment
	}
	ConflictAssignment struct {
		Column string
		Expr   ConflictExpr
	}
	valueConflictExpr struct {
		value any
	}
	incConflictExpr struct {
		column string
		delta  any
	}

	ConflictExpr interface {
		isConflictExpr()
	}

	createBuildResolved struct {
		BuildOpts     builder.InsertBuildOptions
		ReturningCols []mapper.ColumnMeta
		TargetCols    []mapper.ColumnMeta
	}
)

func Value(v any) ConflictExpr {
	return valueConflictExpr{value: v}
}
func Inc(column string, delta any) ConflictExpr {
	return incConflictExpr{column: column, delta: delta}
}

func (valueConflictExpr) isConflictExpr() {}

func (incConflictExpr) isConflictExpr() {}
