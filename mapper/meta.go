package mapper

import "reflect"

type (
	Meta struct {
		Table       string
		Columns     []ColumnMeta
		ColumnIndex map[string]int
	}
	ColumnMeta struct {
		Name       string
		Value      any
		PrimaryKey bool
		FieldSrc   reflect.Value
	}
)

func (m Meta) GetPrimaryKeyColumn() *ColumnMeta {
	for i := range m.Columns {
		if m.Columns[i].PrimaryKey {
			return &m.Columns[i]
		}
	}
	return nil
}
