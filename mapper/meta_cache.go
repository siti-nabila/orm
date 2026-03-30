package mapper

import (
	"reflect"
	"sync"
)

var (
	defaultMetaCache = &metaCacheStore{
		data: make(map[metaCacheKey]*cachedMeta),
	}
)

type (
	metaCacheKey struct {
		modelType    reflect.Type
		useSnakeCase bool
	}
	cachedColumnMeta struct {
		Name       string
		PrimaryKey bool
		Index      []int
	}
	cachedMeta struct {
		Table       string
		Columns     []cachedColumnMeta
		ColumnIndex map[string]int
	}
	metaCacheStore struct {
		mu   sync.RWMutex
		data map[metaCacheKey]*cachedMeta
	}
)
