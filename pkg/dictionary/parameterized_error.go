package dictionary

import (
	_ "embed"

	"github.com/godev90/validator/faults"
)

var (
	errorPack          faults.YamlPackage
	errUnsupportedType error
	//go:embed parameterized_err_list.yaml
	errorList []byte
)

func init() {
	errorPack = faults.NewYamlPackage()
	errorPack.LoadBytes(errorList)

	errUnsupportedType = errorPack.NewError("err_placeholder_not_supported")
}

func UnsupportedTypeError(dialectName string) error {
	return errUnsupportedType.(faults.Error).Render(dialectName)
}
