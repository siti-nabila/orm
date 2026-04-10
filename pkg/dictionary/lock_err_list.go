package dictionary

import (
	_ "embed"

	"github.com/godev90/validator/faults"
)

var (
	errLockPack faults.YamlPackage

	ErrLockConflict           error
	ErrLockEmptyKey           error
	ErrLockUnsupportedDialect error
	ErrLockNotAcquired        error
	//go:embed lock_err_list.yaml
	errLockList []byte
)

func init() {
	errLockPack = faults.NewYamlPackage()
	errLockPack.LoadBytes(errLockList)
	ErrLockConflict = errLockPack.NewError("err_lock_conflict")
	ErrLockEmptyKey = errLockPack.NewError("err_lock_empty_key")
	ErrLockUnsupportedDialect = errLockPack.NewError("err_lock_unsupported_dialect")
	ErrLockNotAcquired = errLockPack.NewError("err_lock_not_acquired")

}
