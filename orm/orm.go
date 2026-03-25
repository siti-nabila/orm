package orm

import (
	"github.com/siti-nabila/orm/builder"
	"github.com/siti-nabila/orm/config"
	"github.com/siti-nabila/orm/db"
	"github.com/siti-nabila/orm/mapper"
	"github.com/siti-nabila/orm/pkg/dictionary"
	"github.com/siti-nabila/orm/pkg/logger"
)

type (
	ORM struct {
		executor db.Executor
		config   config.Config
		logger   logger.Logger
		debug    bool
	}
)

func New(executor db.Executor, config config.Config) *ORM {
	return &ORM{
		executor: executor,
		config:   config,
		debug:    config.EnableDebug,
	}
}

func (o *ORM) SetLogger(l logger.Logger, debug bool) {
	o.logger = l
	o.debug = debug
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
