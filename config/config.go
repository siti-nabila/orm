package config

type (
	PlaceholderMode int
	Config          struct {
		QuoteIdentifier bool
		UseSnakeCase    bool
		PlaceholderMode PlaceholderMode
		EnableDebug     bool
	}
)

const (
	PlaceholderAuto PlaceholderMode = iota
	PlaceholderByNumber
	PlaceholderByName

	QuerySeperator = ", "
)
