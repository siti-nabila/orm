package config

type (
	PlaceholderMode int
	Config          struct {
		QuoteIdentifier bool
		UseSnakeCase    bool
		PlaceholderMode PlaceholderMode
		EnableDebug     bool
		LogDryRunQuery  bool
		LogLockQuery    bool
	}
)

const (
	PlaceholderAuto PlaceholderMode = iota
	PlaceholderByNumber
	PlaceholderByName

	QuerySeperator = ", "
)
