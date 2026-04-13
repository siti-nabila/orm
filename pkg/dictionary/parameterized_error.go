package dictionary

import (
	_ "embed"
	"strings"

	"github.com/godev90/validator/faults"
)

var (
	errorPack                          faults.YamlPackage
	errUnsupportedType                 error
	errColNotFoundOnDest               error
	errUnaddressableDest               error
	errColOverflow                     error
	errNegativeValueUint               error
	errAdvInsConflictUnsupportedAction error
	errScanTypeMismatch                error
	errScanTypeIntMismatch             error
	errScanIntoColCountMismatch        error

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
	errAdvInsConflictUnsupportedAction = errorPack.NewError("err_adv_ins_conflict_unsupported_action")
	errScanTypeMismatch = errorPack.NewError("err_scan_type_mismatch")
	errScanTypeIntMismatch = errorPack.NewError("err_scan_type_int_mismatch")
	errScanIntoColCountMismatch = errorPack.NewError("err_scan_into_col_count_mismatch")
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

func ErrAdvInsConflictUnsupportedAction(actionName string) error {
	return errAdvInsConflictUnsupportedAction.(faults.Error).Render(actionName)
}

func ErrScanTypeMismatch(colName string, value any) error {
	return errScanTypeMismatch.(faults.Error).Render(colName, value)
}

func ErrScanTypeIntMismatch(colName string, value any) error {
	return errScanTypeIntMismatch.(faults.Error).Render(colName, value)
}

func ErrScanIntoColCountMismatch(expected, actual int) error {
	return errScanIntoColCountMismatch.(faults.Error).Render(expected, actual)
}
