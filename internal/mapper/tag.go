package mapper

import (
	"reflect"
	"strings"
)

func parseSQLTag(f reflect.StructField, useSnake bool) (ColumnMeta, bool) {

	tag := f.Tag.Get("sql")

	// skip
	if tag == "-" {
		return ColumnMeta{}, false
	}

	col := ColumnMeta{}

	// no tag -> default
	if tag == "" {
		if useSnake {
			col.Name = toSnake(f.Name)
		} else {
			col.Name = f.Name
		}

		return col, true
	}

	parts := strings.Split(tag, ";")

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p == "-" {
			return ColumnMeta{}, false
		}

		if p == "primaryKey" {
			col.PrimaryKey = true
			continue
		}

		if strings.HasPrefix(p, "column:") {
			col.Name = strings.TrimPrefix(p, "column:")
			continue
		}
	}

	if col.Name == "" {
		if useSnake {
			col.Name = toSnake(f.Name)
		} else {
			col.Name = f.Name
		}
	}

	return col, true

}
