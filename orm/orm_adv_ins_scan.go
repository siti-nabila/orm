package orm

import (
	"github.com/siti-nabila/orm/dialect"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

func prepareAdvInsScanTargets(
	cols []mapper.ColumnMeta,
	d dialect.Dialector,
) ([]any, []scanAssignment, error) {
	if len(cols) == 0 {
		return nil, nil, dictionary.ErrDBQueryEmpty
	}

	targets := make([]any, 0, len(cols))
	assignments := make([]scanAssignment, 0)

	for _, col := range cols {
		field := col.FieldSrc

		if !field.IsValid() || !field.CanAddr() {
			return nil, nil, dictionary.ErrUnaddressableDestError(col.Name)
		}

		target, assignment, err := buildScanTargetForField(d, col.Name, field)
		if err != nil {
			return nil, nil, err
		}

		targets = append(targets, target)
		if assignment != nil {
			assignments = append(assignments, *assignment)
		}
	}

	return targets, assignments, nil
}

func buildReturningScanTargets(cols []mapper.ColumnMeta) ([]any, error) {
	targets := make([]any, 0, len(cols))

	for _, col := range cols {
		if !col.FieldSrc.IsValid() || !col.FieldSrc.CanAddr() {
			return nil, dictionary.ErrDBScanUnsupportedDest
		}
		targets = append(targets, col.FieldSrc.Addr().Interface())
	}

	return targets, nil
}
