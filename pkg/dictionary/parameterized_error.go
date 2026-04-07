package dictionary

import (
	_ "embed"
	"strings"

	"github.com/godev90/validator/faults"
)

var (
	errorPack            faults.YamlPackage
	errUnsupportedType   error
	errColNotFoundOnDest error
	errUnaddressableDest error
	errColOverflow       error
	errNegativeValueUint error

	//go:embed parameterized_err_list.yaml
	errorList []byte
)

func init() {
	errorPack = faults.NewYamlPackage()
	errorPack.LoadBytes(errorList)

	errUnsupportedType = errorPack.NewError("err_placeholder_not_supported")
	errColNotFoundOnDest = errorPack.NewError("err_col_not_found_on_dest")
	errUnaddressableDest = errorPack.NewError("err_unaddressable_dest")
	errColOverflow = errorPack.NewError("err_col_overflow")
	errNegativeValueUint = errorPack.NewError("err_negative_value_uint")
}

func UnsupportedTypeError(dialectName string) error {
	return errUnsupportedType.(faults.Error).Render(dialectName)
}

func ErrColNotFoundOnDestError(cols []string) error {
	colNames := strings.Join(cols, ", ")
	return errColNotFoundOnDest.(faults.Error).Render(colNames)
}

func ErrUnaddressableDestError(colName string) error {
	return errUnaddressableDest.(faults.Error).Render(colName)
}

func ErrColOverflowError(colName string, value any) error {
	return errColOverflow.(faults.Error).Render(colName, value)
}

func ErrNegativeValueUintError(colName string, value int64) error {
	return errNegativeValueUint.(faults.Error).Render(colName, value)
}
