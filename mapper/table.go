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

func getTableNameFromModelType(typ reflect.Type, useSnake bool) (string, error) {
	tablerType := reflect.TypeOf((*Tabler)(nil)).Elem()

	if typ.Implements(tablerType) {
		v := reflect.New(typ).Elem().Interface().(Tabler)
		return v.TableName(), nil
	}

	if reflect.PointerTo(typ).Implements(tablerType) {
		v := reflect.New(typ).Interface().(Tabler)
		return v.TableName(), nil
	}

	if useSnake {
		return toSnake(typ.Name()), nil
	}

	return typ.Name(), nil
}
