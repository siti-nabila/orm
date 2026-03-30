package dictionary

import (
	_ "embed"

	"github.com/godev90/validator/faults"
)

var (
	errPack faults.YamlPackage

	ErrDBConn                  error
	ErrDBPlaceholder           error
	ErrDBQueryEmpty            error
	ErrDuplicateRow            error
	ErrRowNotFound             error
	ErrDBUnknown               error
	ErrForeignKey              error
	ErrDBTooManyArguments      error
	ErrPrimaryKeyNotFound      error
	ErrPrimaryKeyEmpty         error
	ErrColumnNotFound          error
	ErrDBScanNilDest           error
	ErrDBScanNotPointerDest    error
	ErrDBScanUnsupportedDest   error
	ErrDBScanUnimplemented     error
	ErrDBScanMetaNil           error
	ErrDBScanMustBeSliceStruct error
	ErrInvalidValue            error
	ErrMustBeStructPtr         error

	//go:embed err_list.yaml

	errList []byte
)

func init() {
	errPack = faults.NewYamlPackage()
	errPack.LoadBytes(errList)

	ErrDBConn = errPack.NewError("err_db_conn")
	ErrDBPlaceholder = errPack.NewError("err_db_placeholder")
	ErrDBQueryEmpty = errPack.NewError("err_db_query_empty")
	ErrDuplicateRow = errPack.NewError("err_duplicate_row")
	ErrRowNotFound = errPack.NewError("err_row_not_found")
	ErrDBUnknown = errPack.NewError("err_db_unknown")
	ErrForeignKey = errPack.NewError("err_foreign_key")
	ErrDBTooManyArguments = errPack.NewError("err_db_too_many_arguments")
	ErrPrimaryKeyNotFound = errPack.NewError("err_pk_not_found")
	ErrPrimaryKeyEmpty = errPack.NewError("err_pk_empty")
	ErrColumnNotFound = errPack.NewError("err_column_not_found")
	ErrDBScanNilDest = errPack.NewError("err_scan_dest_nil")
	ErrDBScanNotPointerDest = errPack.NewError("err_scan_dest_not_pointer")
	ErrDBScanUnsupportedDest = errPack.NewError("err_scan_unsupported_dest")
	ErrDBScanUnimplemented = errPack.NewError("err_scan_unimplemented")
	ErrDBScanMetaNil = errPack.NewError("err_scan_meta_nil")
	ErrDBScanMustBeSliceStruct = errPack.NewError("err_scan_must_be_slice_struct")
	ErrInvalidValue = errPack.NewError("err_invalid_value")
	ErrMustBeStructPtr = errPack.NewError("err_must_be_pointer_struct")
}
