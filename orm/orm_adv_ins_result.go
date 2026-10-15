package orm

import (
	"database/sql"
	"time"

	"github.com/siti-nabila/orm/pkg/dictionary"
)

// ExecResult executes an insert without RETURNING or generated-ID assignment.
// The driver result (including nil) and error are preserved for the caller.
func (c *CreateCommand) ExecResult() (result sql.Result, err error) {
	if c == nil || c.orm == nil {
		return nil, dictionary.ErrDBQueryEmpty
	}
	if err = c.validateReturningMode(false); err != nil {
		return nil, err
	}
	start := time.Now()
	res, _, _, d, err := c.prepareCreateWith()
	if err != nil {
		return nil, err
	}
	defer func() { c.orm.log(res.Query, d, res.FilteredCols, res.Args, res.Mode, start, err) }()
	return c.orm.executor.ExecContext(c.ctx, res.Query, res.Args...)
}
