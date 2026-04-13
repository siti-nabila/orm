package orm

import (
	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

func resolveCreateBuildOptions(
	meta *mapper.Meta,
	opts CreateOptions,
	d dialect.Dialector,
) (createBuildResolved, error) {
	returningCols, err := validateReturningColumns(meta, opts.Returning)
	if err != nil {
		return createBuildResolved{}, err
	}

	conflictClause, targetCols, err := resolveOnConflict(meta, opts.OnConflict, d)
	if err != nil {
		return createBuildResolved{}, err
	}

	return createBuildResolved{
		BuildOpts: builder.InsertBuildOptions{
			ReturningCols: returningCols,
			OnConflict:    conflictClause,
		},
		ReturningCols: returningCols,
		TargetCols:    targetCols,
	}, nil
}

func validateReturningColumns(
	meta *mapper.Meta,
	cols []string,
) ([]mapper.ColumnMeta, error) {
	if len(cols) == 0 {
		return nil, nil
	}

	out := make([]mapper.ColumnMeta, 0, len(cols))
	seen := make(map[string]struct{}, len(cols))

	for _, colName := range cols {
		if colName == "" {
			return nil, dictionary.ErrAdvInsReturningNotFound
		}
		if _, ok := seen[colName]; ok {
			continue
		}
		seen[colName] = struct{}{}

		colMeta, ok := findMetaColumn(meta, colName)
		if !ok {
			return nil, dictionary.ErrAdvInsReturningNotFound
		}

		out = append(out, colMeta)
	}

	return out, nil
}

func resolveOnConflict(
	meta *mapper.Meta,
	conf *OnConflict,
	d dialect.Dialector,
) (*builder.OnConflictClause, []mapper.ColumnMeta, error) {
	if conf == nil {
		return nil, nil, nil
	}

	if len(conf.TargetColumns) == 0 {
		return nil, nil, dictionary.ErrAdvInsTargetColumnEmpty
	}

	if conf.DoNothing {
		switch d.Type() {
		case dialect.DialectPostgres:
		case dialect.DialectMySQL, dialect.DialectOracle:
			return nil, nil, dictionary.ErrAdvInsConflictUnsupportedAction("do_nothing")
		default:
			return nil, nil, dictionary.ErrAdvInsConflictUnsupportedAction("do_nothing")
		}
	}

	targetCols, err := resolveConflictTargetColumns(meta, conf.TargetColumns)
	if err != nil {
		return nil, nil, err
	}

	assignments, err := resolveConflictAssignments(meta, conf)
	if err != nil {
		return nil, nil, err
	}

	clause := &builder.OnConflictClause{
		TargetCols:  targetCols,
		DoNothing:   conf.DoNothing,
		Assignments: assignments,
	}

	return clause, targetCols, nil
}

func resolveConflictTargetColumns(
	meta *mapper.Meta,
	targetCols []string,
) ([]mapper.ColumnMeta, error) {
	out := make([]mapper.ColumnMeta, 0, len(targetCols))
	seen := make(map[string]struct{}, len(targetCols))

	for _, colName := range targetCols {
		if colName == "" {
			return nil, dictionary.ErrAdvInsConflictTargetColumnNotFound
		}
		if _, ok := seen[colName]; ok {
			continue
		}
		seen[colName] = struct{}{}

		colMeta, ok := findMetaColumn(meta, colName)
		if !ok {
			return nil, dictionary.ErrAdvInsConflictTargetColumnNotFound
		}

		out = append(out, colMeta)
	}

	return out, nil
}

func resolveConflictAssignments(
	meta *mapper.Meta,
	conf *OnConflict,
) ([]builder.ResolvedConflictAssignment, error) {
	total := len(conf.DoUpdates) + len(conf.Assignments)

	if conf.DoNothing {
		if total > 0 {
			return nil, dictionary.ErrAdvInsConflictDoNothingDoUpdateUnsupported
		}
		return nil, nil
	}

	if total == 0 {
		return nil, dictionary.ErrAdvInsConflictNoAction
	}

	seen := make(map[string]struct{}, total)
	out := make([]builder.ResolvedConflictAssignment, 0, total)

	doUpdates, err := resolveDoUpdateAssignments(meta, conf.DoUpdates, seen)
	if err != nil {
		return nil, err
	}
	out = append(out, doUpdates...)

	customAssignments, err := resolveCustomConflictAssignments(meta, conf.Assignments, seen)
	if err != nil {
		return nil, err
	}
	out = append(out, customAssignments...)

	return out, nil
}

func resolveDoUpdateAssignments(
	meta *mapper.Meta,
	cols []string,
	seen map[string]struct{},
) ([]builder.ResolvedConflictAssignment, error) {
	out := make([]builder.ResolvedConflictAssignment, 0, len(cols))

	for _, colName := range cols {
		if colName == "" {
			return nil, dictionary.ErrAdvInsConflictUpdateColumnNotFound
		}
		if _, ok := seen[colName]; ok {
			return nil, dictionary.ErrAdvInsConflictDuplicateAssignment
		}

		colMeta, ok := findMetaColumn(meta, colName)
		if !ok {
			return nil, dictionary.ErrAdvInsConflictUpdateColumnNotFound
		}

		seen[colName] = struct{}{}
		out = append(out, builder.ResolvedConflictAssignment{
			ColumnMeta: colMeta,
			Mode:       builder.ConflictAssignInserted,
		})
	}

	return out, nil
}

func resolveCustomConflictAssignments(
	meta *mapper.Meta,
	assignments []ConflictAssignment,
	seen map[string]struct{},
) ([]builder.ResolvedConflictAssignment, error) {
	out := make([]builder.ResolvedConflictAssignment, 0, len(assignments))

	for _, assignment := range assignments {
		if assignment.Column == "" {
			return nil, dictionary.ErrAdvInsConflictAssignmentColumnNotFound
		}
		if _, ok := seen[assignment.Column]; ok {
			return nil, dictionary.ErrAdvInsConflictDuplicateAssignment
		}

		colMeta, ok := findMetaColumn(meta, assignment.Column)
		if !ok {
			return nil, dictionary.ErrAdvInsConflictAssignmentColumnNotFound
		}

		resolved, err := resolveConflictExpr(meta, colMeta, assignment.Expr)
		if err != nil {
			return nil, err
		}

		seen[assignment.Column] = struct{}{}
		out = append(out, resolved)
	}

	return out, nil
}

func resolveConflictExpr(
	meta *mapper.Meta,
	targetCol mapper.ColumnMeta,
	expr ConflictExpr,
) (builder.ResolvedConflictAssignment, error) {
	switch x := expr.(type) {
	case valueConflictExpr:
		return builder.ResolvedConflictAssignment{
			ColumnMeta: targetCol,
			Mode:       builder.ConflictAssignValue,
			Value:      x.value,
		}, nil

	case incConflictExpr:
		if x.column == "" {
			return builder.ResolvedConflictAssignment{}, dictionary.ErrAdvInsConflictRefColumnNotFound
		}

		refCol, ok := findMetaColumn(meta, x.column)
		if !ok {
			return builder.ResolvedConflictAssignment{}, dictionary.ErrAdvInsConflictRefColumnNotFound
		}

		return builder.ResolvedConflictAssignment{
			ColumnMeta: targetCol,
			Mode:       builder.ConflictAssignInc,
			Value:      x.delta,
			RefColumn:  &refCol,
		}, nil

	default:
		return builder.ResolvedConflictAssignment{}, dictionary.ErrAdvInsInvalidMode
	}
}

func findMetaColumn(
	meta *mapper.Meta,
	colName string,
) (mapper.ColumnMeta, bool) {
	if meta == nil {
		return mapper.ColumnMeta{}, false
	}

	if meta.ColumnIndex != nil {
		if idx, ok := meta.ColumnIndex[colName]; ok {
			if idx >= 0 && idx < len(meta.Columns) {
				return meta.Columns[idx], true
			}
		}
	}

	for _, col := range meta.Columns {
		if col.Name == colName {
			return col, true
		}
	}

	return mapper.ColumnMeta{}, false
}
