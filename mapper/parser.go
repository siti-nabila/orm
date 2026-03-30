package mapper

import (
	"reflect"

	"github.com/siti-nabila/orm/pkg/dictionary"
)

func Parse(v any, useSnake bool) (*Meta, error) {
	val := reflect.ValueOf(v)
	if !val.IsValid() {
		return nil, dictionary.ErrInvalidValue
	}

	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil, dictionary.ErrInvalidValue
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, dictionary.ErrInvalidValue
	}

	typ := val.Type()

	cached, err := getOrCreateCachedMeta(typ, useSnake)
	if err != nil {
		return nil, err
	}

	meta := &Meta{
		Table:       cached.Table,
		Columns:     make([]ColumnMeta, 0, len(cached.Columns)),
		ColumnIndex: cached.ColumnIndex,
	}

	for _, cachedCol := range cached.Columns {
		fVal := val.FieldByIndex(cachedCol.Index)

		meta.Columns = append(meta.Columns, ColumnMeta{
			Name:       cachedCol.Name,
			Value:      fVal.Interface(),
			PrimaryKey: cachedCol.PrimaryKey,
			FieldSrc:   fVal,
		})
	}

	return meta, nil
}

func getOrCreateCachedMeta(modelType reflect.Type, useSnake bool) (*cachedMeta, error) {
	for modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	key := metaCacheKey{
		modelType:    modelType,
		useSnakeCase: useSnake,
	}

	defaultMetaCache.mu.RLock()
	if meta, ok := defaultMetaCache.data[key]; ok {
		defaultMetaCache.mu.RUnlock()
		return meta, nil
	}
	defaultMetaCache.mu.RUnlock()

	meta, err := parseCachedMeta(modelType, useSnake)
	if err != nil {
		return nil, err
	}

	defaultMetaCache.mu.Lock()
	defer defaultMetaCache.mu.Unlock()

	if existing, ok := defaultMetaCache.data[key]; ok {
		return existing, nil
	}

	defaultMetaCache.data[key] = meta
	return meta, nil
}

func parseCachedMeta(modelType reflect.Type, useSnake bool) (*cachedMeta, error) {
	if modelType.Kind() != reflect.Struct {
		return nil, dictionary.ErrInvalidValue
	}

	table, err := getTableNameFromModelType(modelType, useSnake)
	if err != nil {
		return nil, err
	}

	meta := &cachedMeta{
		Table:       table,
		Columns:     make([]cachedColumnMeta, 0, modelType.NumField()),
		ColumnIndex: make(map[string]int, modelType.NumField()),
	}

	for i := 0; i < modelType.NumField(); i++ {
		sf := modelType.Field(i)

		col, include, err := parseStructField(sf, useSnake)
		if err != nil {
			return nil, err
		}
		if !include {
			continue
		}
		colIdx := len(meta.Columns)

		meta.Columns = append(meta.Columns, cachedColumnMeta{
			Name:       col.Name,
			PrimaryKey: col.PrimaryKey,
			Index:      sf.Index,
		})
		meta.ColumnIndex[col.Name] = colIdx
	}

	return meta, nil
}

func parseStructField(field reflect.StructField, useSnake bool) (ColumnMeta, bool, error) {
	// skip embedded field
	if field.Anonymous {
		return ColumnMeta{}, false, nil
	}
	// skip unexported non-embedded field
	if !field.IsExported() {
		return ColumnMeta{}, false, nil
	}

	col, ok := parseSQLTag(field, useSnake)
	if !ok {
		return ColumnMeta{}, false, nil
	}
	return col, true, nil
}
