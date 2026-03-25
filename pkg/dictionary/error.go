package dictionary

import (
	_ "embed"

	"github.com/godev90/validator/faults"
)

var (
	errPack faults.YamlPackage

	ErrDBConn        error
	ErrDBPlaceholder error
	ErrDBQueryEmpty  error
	//go:embed err_list.yaml

	errList []byte
)

func init() {
	errPack = faults.NewYamlPackage()
	errPack.LoadBytes(errList)

	ErrDBConn = errPack.NewError("err_db_conn")
	ErrDBPlaceholder = errPack.NewError("err_db_placeholder")
	ErrDBQueryEmpty = errPack.NewError("err_db_query_empty")
}
