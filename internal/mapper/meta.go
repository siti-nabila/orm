package mapper

import "reflect"

type (
	Meta struct {
		Table   string
		Columns []ColumnMeta
	}
	ColumnMeta struct {
		Name       string
		Value      any
		PrimaryKey bool
		FieldSrc   reflect.Value
	}
)

func (m Meta) GetPrimaryKeyColumn() *ColumnMeta {
	for _, col := range m.Columns {
		if col.PrimaryKey {
			return &col
		}
	}
	return nil
}
