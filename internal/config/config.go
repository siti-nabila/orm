package config

type (
	PlaceholderMode int
	Config          struct {
		QuoteIdentifier bool
		UseSnakeCase    bool
		PlaceholderMode PlaceholderMode
	}
)

const (
	PlaceholderAuto PlaceholderMode = iota
	PlaceholderByNumber
	PlaceholderByName

	QuerySeperator = ", "
)
