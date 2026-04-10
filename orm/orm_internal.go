package orm

func (o *ORM) shouldLogLockQuery() bool {
	return o != nil && o.config.LogLockQuery
}
