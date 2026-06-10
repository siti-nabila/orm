package dictionary

import (
	_ "embed"

	// "github.com/godev90/validator/faults"
	errorpackage "github.com/siti-nabila/error-package"
)

var (
	// errLockPack faults.YamlPackage
	errLockPack errorpackage.DictionaryPack

	ErrLockConflict           error
	ErrLockEmptyKey           error
	ErrLockUnsupportedDialect error
	ErrLockNotAcquired        error
	//go:embed lock_err_list.yaml
	errLockList []byte
)

func init() {
	// errLockPack = faults.NewYamlPackage()
	// errLockPack.LoadBytes(errLockList)
	errLockPack = errorpackage.NewErrYamlPackage()
	if err := errLockPack.LoadBytes(errLockList); err != nil {
		panic(err)
	}
	// ErrLockConflict = errLockPack.NewError("err_lock_conflict")
	ErrLockConflict = errLockPack.New("err_lock_conflict")
	// ErrLockEmptyKey = errLockPack.NewError("err_lock_empty_key")
	ErrLockEmptyKey = errLockPack.New("err_lock_empty_key")
	// ErrLockUnsupportedDialect = errLockPack.NewError("err_lock_unsupported_dialect")
	ErrLockUnsupportedDialect = errLockPack.New("err_lock_unsupported_dialect")
	// ErrLockNotAcquired = errLockPack.NewError("err_lock_not_acquired")
	ErrLockNotAcquired = errLockPack.New("err_lock_not_acquired")

}
