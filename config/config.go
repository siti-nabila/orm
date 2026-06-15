package config

import "github.com/siti-nabila/orm/pagination"

type (
	PlaceholderMode int
	Config          struct {
		QuoteIdentifier bool
		UseSnakeCase    bool
		PlaceholderMode PlaceholderMode
		EnableDebug     bool
		LogDryRunQuery  bool
		LogLockQuery    bool
		Pagination      pagination.Config
	}
)

const (
	PlaceholderAuto PlaceholderMode = iota
	PlaceholderByNumber
	PlaceholderByName

	QuerySeperator = ", "
)
