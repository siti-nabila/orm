package dictionary

import (
	_ "embed"

	// "github.com/godev90/validator/faults"
	errorpackage "github.com/siti-nabila/error-package"
)

var (
	// errPack faults.YamlPackage
	errPack errorpackage.DictionaryPack

	ErrDBConn                          error
	ErrDBPlaceholder                   error
	ErrDBQueryEmpty                    error
	ErrDuplicateRow                    error
	ErrRowNotFound                     error
	ErrDBUnknown                       error
	ErrForeignKey                      error
	ErrDBTooManyArguments              error
	ErrPrimaryKeyNotFound              error
	ErrPrimaryKeyEmpty                 error
	ErrColumnNotFound                  error
	ErrDBScanNilDest                   error
	ErrDBScanNotPointerDest            error
	ErrDBScanUnsupportedDest           error
	ErrDBScanUnimplemented             error
	ErrDBScanMetaNil                   error
	ErrDBScanMustBeSliceStruct         error
	ErrInvalidValue                    error
	ErrMustBeStructPtr                 error
	ErrMustBeSlicePtr                  error
	ErrDBScanPrimitiveMustSingleColumn error
	ErrDBScanIntoEmptyDest             error
	ErrPaginationInvalidLimit          error
	ErrPaginationOffsetOverflow        error
	ErrPaginationTotalWithJoin         error
	ErrPaginationTotalUnsupported      error
	ErrPaginationTotalColumnNotFound   error

	// error bulk insert
	ErrBulkInsertElemNil                    error
	ErrBulkInsertElemNotStruct              error
	ErrBulkInsertElemTypeMismatch           error
	ErrBulkInsertValueNil                   error
	ErrBulkInsertValueNotPointerSlice       error
	ErrBulkInsertValueSliceElementNotStruct error
	ErrBulkInsertValueEmpty                 error
	ErrBulkInsertTableMismatch              error
	ErrBulkInsertPrimaryKeyMismatch         error
	ErrBulkInsertColumnMismatch             error
	ErrBulkInsertColumnCountMismatch        error
	ErrBulkInsertEmptyMetas                 error
	ErrUnsupportedDialect                   error
	// error bulk update
	ErrBulkUpdateEmptyMetas         error
	ErrBulkUpdatePrimaryKeyMismatch error
	ErrBulkUpdateColumnMismatch     error
	// error advanced insert
	ErrAdvInsIncMissingRefColumn                  error
	ErrAdvInsInvalidMode                          error
	ErrAdvInsReturningNotFound                    error
	ErrAdvInsTargetColumnEmpty                    error
	ErrAdvInsConflictNoAction                     error
	ErrAdvInsConflictDoNothingDoUpdateUnsupported error
	ErrAdvInsConflictTargetColumnNotFound         error
	ErrAdvInsConflictUpdateColumnNotFound         error
	ErrAdvInsConflictAssignmentColumnNotFound     error
	ErrAdvInsConflictRefColumnNotFound            error
	ErrAdvInsConflictDuplicateAssignment          error
	ErrAdvInsMySQLScanRequiresTarget              error
	ErrAdvInsScanWithoutReturning                 error
	ErrAdvInsExecWithReturning                    error
	ErrAdvInsOracleReturningBindFailed            error

	//go:embed err_list.yaml

	errList []byte
)

func init() {
	// errPack = faults.NewYamlPackage()
	// errPack.LoadBytes(errList)
	errPack = errorpackage.NewErrYamlPackage()
	if err := errPack.LoadBytes(errList); err != nil {
		panic(err)
	}

	// ErrDBConn = errPack.NewError("err_db_conn")
	ErrDBConn = errPack.New("err_db_conn")
	// ErrDBPlaceholder = errPack.NewError("err_db_placeholder")
	ErrDBPlaceholder = errPack.New("err_db_placeholder")
	// ErrDBQueryEmpty = errPack.NewError("err_db_query_empty")
	ErrDBQueryEmpty = errPack.New("err_db_query_empty")
	// ErrDuplicateRow = errPack.NewError("err_duplicate_row")
	ErrDuplicateRow = errPack.New("err_duplicate_row")
	// ErrRowNotFound = errPack.NewError("err_row_not_found")
	ErrRowNotFound = errPack.New("err_row_not_found")
	// ErrDBUnknown = errPack.NewError("err_db_unknown")
	ErrDBUnknown = errPack.New("err_db_unknown")
	// ErrForeignKey = errPack.NewError("err_foreign_key")
	ErrForeignKey = errPack.New("err_foreign_key")
	// ErrDBTooManyArguments = errPack.NewError("err_db_too_many_arguments")
	ErrDBTooManyArguments = errPack.New("err_db_too_many_arguments")
	// ErrPrimaryKeyNotFound = errPack.NewError("err_pk_not_found")
	ErrPrimaryKeyNotFound = errPack.New("err_pk_not_found")
	// ErrPrimaryKeyEmpty = errPack.NewError("err_pk_empty")
	ErrPrimaryKeyEmpty = errPack.New("err_pk_empty")
	// ErrColumnNotFound = errPack.NewError("err_column_not_found")
	ErrColumnNotFound = errPack.New("err_column_not_found")
	// ErrDBScanNilDest = errPack.NewError("err_scan_dest_nil")
	ErrDBScanNilDest = errPack.New("err_scan_dest_nil")
	// ErrDBScanNotPointerDest = errPack.NewError("err_scan_dest_not_pointer")
	ErrDBScanNotPointerDest = errPack.New("err_scan_dest_not_pointer")
	// ErrDBScanUnsupportedDest = errPack.NewError("err_scan_unsupported_dest")
	ErrDBScanUnsupportedDest = errPack.New("err_scan_unsupported_dest")
	// ErrDBScanUnimplemented = errPack.NewError("err_scan_unimplemented")
	ErrDBScanUnimplemented = errPack.New("err_scan_not_implemented")
	// ErrDBScanMetaNil = errPack.NewError("err_scan_meta_nil")
	ErrDBScanMetaNil = errPack.New("err_meta_nil")
	// ErrDBScanMustBeSliceStruct = errPack.NewError("err_scan_must_be_slice_struct")
	ErrDBScanMustBeSliceStruct = errPack.New("err_scan_must_slice_struct")
	// ErrInvalidValue = errPack.NewError("err_invalid_value")
	ErrInvalidValue = errPack.New("err_invalid_value")
	// ErrMustBeStructPtr = errPack.NewError("err_must_be_pointer_struct")
	ErrMustBeStructPtr = errPack.New("err_must_be_pointer_struct")
	// ErrMustBeSlicePtr = errPack.NewError("err_must_be_pointer_slice")
	ErrMustBeSlicePtr = errPack.New("err_must_be_pointer_slice")
	// ErrDBScanPrimitiveMustSingleColumn = errPack.NewError("err_scan_primitive_must_single_column")
	ErrDBScanPrimitiveMustSingleColumn = errPack.New("err_db_scan_primitive_must_single_column")
	// ErrDBScanIntoEmptyDest = errPack.NewError("err_scan_into_empty_dest")
	ErrDBScanIntoEmptyDest = errPack.New("err_db_scan_into_empty_dest")
	// pagination errors
	ErrPaginationInvalidLimit = errPack.New("err_pagination_invalid_limit")
	ErrPaginationOffsetOverflow = errPack.New("err_pagination_offset_overflow")
	ErrPaginationTotalWithJoin = errPack.New("err_pagination_total_with_join")
	ErrPaginationTotalUnsupported = errPack.New("err_pagination_total_unsupported")
	ErrPaginationTotalColumnNotFound = errPack.New("err_pagination_total_column_not_found")
	// error bulk insert
	// ErrBulkInsertElemNil = errPack.NewError("err_bulk_insert_elem_nil")
	ErrBulkInsertElemNil = errPack.New("err_bulk_insert_element_nil")
	// ErrBulkInsertElemNotStruct = errPack.NewError("err_bulk_insert_elem_not_struct")
	ErrBulkInsertElemNotStruct = errPack.New("err_bulk_insert_element_not_struct")
	// ErrBulkInsertElemTypeMismatch = errPack.NewError("err_bulk_insert_elem_type_mismatch")
	ErrBulkInsertElemTypeMismatch = errPack.New("err_bulk_insert_element_type_mismatch")
	// ErrBulkInsertValueNil = errPack.NewError("err_bulk_insert_value_nil")
	ErrBulkInsertValueNil = errPack.New("err_bulk_insert_value_nil")
	// ErrBulkInsertValueNotPointerSlice = errPack.NewError("err_bulk_insert_value_not_pointer_slice")
	ErrBulkInsertValueNotPointerSlice = errPack.New("err_bulk_insert_value_not_pointer_slice")
	// ErrBulkInsertValueSliceElementNotStruct = errPack.NewError("err_bulk_insert_value_slice_element_not_struct")
	ErrBulkInsertValueSliceElementNotStruct = errPack.New("err_bulk_insert_value_slice_element_not_struct")
	// ErrBulkInsertValueEmpty = errPack.NewError("err_bulk_insert_value_empty")
	ErrBulkInsertValueEmpty = errPack.New("err_bulk_insert_value_empty")
	// ErrBulkInsertTableMismatch = errPack.NewError("err_bulk_insert_table_mismatch")
	ErrBulkInsertTableMismatch = errPack.New("err_bulk_insert_table_mismatch")
	// ErrBulkInsertPrimaryKeyMismatch = errPack.NewError("err_bulk_insert_primary_key_mismatch")
	ErrBulkInsertPrimaryKeyMismatch = errPack.New("err_bulk_insert_primary_key_mismatch")
	// ErrBulkInsertColumnMismatch = errPack.NewError("err_bulk_insert_column_mismatch")
	ErrBulkInsertColumnMismatch = errPack.New("err_bulk_insert_column_mismatch")
	// ErrBulkInsertColumnCountMismatch = errPack.NewError("err_bulk_insert_column_count_mismatch")
	ErrBulkInsertColumnCountMismatch = errPack.New("err_bulk_insert_column_count_mismatch")
	// ErrBulkInsertEmptyMetas = errPack.NewError("err_bulk_insert_empty_metas")
	ErrBulkInsertEmptyMetas = errPack.New("err_bulk_insert_empty_metas")
	// ErrUnsupportedDialect = errPack.NewError("err_unsupported_dialect")
	ErrUnsupportedDialect = errPack.New("err_unsupported_dialect")
	// error bulk update
	ErrBulkUpdateEmptyMetas = errPack.New("err_bulk_update_empty_metas")
	ErrBulkUpdatePrimaryKeyMismatch = errPack.New("err_bulk_update_primary_key_mismatch")
	ErrBulkUpdateColumnMismatch = errPack.New("err_bulk_update_column_mismatch")
	// error advanced insert
	// ErrAdvInsIncMissingRefColumn = errPack.NewError("err_adv_ins_inc_missing_ref_column")
	ErrAdvInsIncMissingRefColumn = errPack.New("err_adv_ins_inc_missing_ref_column")
	// ErrAdvInsInvalidMode = errPack.NewError("err_adv_ins_invalid_mode")
	ErrAdvInsInvalidMode = errPack.New("err_adv_ins_invalid_mode")
	// ErrAdvInsReturningNotFound = errPack.NewError("err_adv_ins_returning_not_found")
	ErrAdvInsReturningNotFound = errPack.New("err_adv_ins_returning_not_found")
	// ErrAdvInsTargetColumnEmpty = errPack.NewError("err_adv_ins_target_column_empty")
	ErrAdvInsTargetColumnEmpty = errPack.New("err_adv_ins_target_col_empty")
	// ErrAdvInsConflictNoAction = errPack.NewError("err_adv_ins_conflict_no_action")
	ErrAdvInsConflictNoAction = errPack.New("err_adv_ins_conflict_no_action")
	// ErrAdvInsConflictDoNothingDoUpdateUnsupported = errPack.NewError("err_adv_ins_conflict_donothing_with_update")
	ErrAdvInsConflictDoNothingDoUpdateUnsupported = errPack.New("err_adv_ins_conflict_donothing_with_update")
	// ErrAdvInsConflictTargetColumnNotFound = errPack.NewError("err_adv_ins_conflict_target_column_not_found")
	ErrAdvInsConflictTargetColumnNotFound = errPack.New("err_adv_ins_conflict_target_col_not_found")
	// ErrAdvInsConflictUpdateColumnNotFound = errPack.NewError("err_adv_ins_conflict_update_column_not_found")
	ErrAdvInsConflictUpdateColumnNotFound = errPack.New("err_adv_ins_conflict_update_col_not_found")
	// ErrAdvInsConflictAssignmentColumnNotFound = errPack.NewError("err_adv_ins_conflict_assignment_column_not_found")
	ErrAdvInsConflictAssignmentColumnNotFound = errPack.New("err_adv_ins_conflict_assignment_col_not_found")
	// ErrAdvInsConflictRefColumnNotFound = errPack.NewError("err_adv_ins_conflict_ref_column_not_found")
	ErrAdvInsConflictRefColumnNotFound = errPack.New("err_adv_ins_conflict_ref_col_not_found")
	// ErrAdvInsConflictDuplicateAssignment = errPack.NewError("err_adv_ins_conflict_duplicate_assignment")
	ErrAdvInsConflictDuplicateAssignment = errPack.New("err_adv_ins_conflict_duplicate_assignment")
	// ErrAdvInsMySQLScanRequiresTarget = errPack.NewError("err_adv_ins_mysql_scan_requires_target")
	ErrAdvInsMySQLScanRequiresTarget = errPack.New("err_adv_ins_mysql_scan_requires_target")
	// ErrAdvInsScanWithoutReturning = errPack.NewError("err_adv_ins_scan_without_returning")
	ErrAdvInsScanWithoutReturning = errPack.New("err_adv_ins_scan_without_returning")
	// ErrAdvInsExecWithReturning = errPack.NewError("err_adv_ins_exec_with_returning")
	ErrAdvInsExecWithReturning = errPack.New("err_adv_ins_exec_with_returning")
	// ErrAdvInsOracleReturningBindFailed = errPack.NewError("err_adv_ins_oracle_returning_bind_failed")
	ErrAdvInsOracleReturningBindFailed = errPack.New("err_adv_ins_oracle_returning_bind_failed")
}
