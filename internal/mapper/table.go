package mapper

import "reflect"

type (
	Tabler interface {
		TableName() string
	}
)

func getTableName(
	v any,
	t reflect.Type,
	useSnake bool,
) string {

	if tabler, ok := v.(Tabler); ok {
		return tabler.TableName()
	}

	if useSnake {
		return toSnake(t.Name())
	}

	return t.Name()
}
