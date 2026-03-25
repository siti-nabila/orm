package orm

import (
	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/db"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
)

type (
	ORM struct {
		executor db.Executor
		config   config.Config
	}
)

func New(executor db.Executor, config config.Config) *ORM {
	return &ORM{
		executor: executor,
		config:   config,
	}
}

func (o *ORM) GeneratePlaceholder(cols []mapper.ColumnMeta) string {
	mode := o.placeholderMode()
	return builder.GeneratePlaceholderQuery(o.executor.Dialect(), mode, cols)
}

func (o *ORM) GenerateColumnList(cols []mapper.ColumnMeta) string {
	quote := o.config.QuoteIdentifier
	return builder.GenerateColumnListQuery(o.executor.Dialect(), quote, cols)
}

func (o *ORM) placeholderMode() config.PlaceholderMode {
	switch o.config.PlaceholderMode {
	case config.PlaceholderByNumber:
		return config.PlaceholderByNumber
	case config.PlaceholderByName:
		return config.PlaceholderByName
	case config.PlaceholderAuto:
		return o.placeholderAutoMode()
	default:
		panic(dictionary.ErrDBPlaceholder)
	}
}

func (o *ORM) placeholderAutoMode() config.PlaceholderMode {
	switch o.executor.Dialect().Name() {
	case "oracle":
		return config.PlaceholderByName
	default:
		return config.PlaceholderByNumber
	}
}
