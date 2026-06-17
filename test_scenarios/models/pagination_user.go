package models

type PaginationUser struct {
	ID     int64  `sql:"column:id;primaryKey"`
	Name   string `sql:"column:name"`
	Active bool   `sql:"column:active"`
}

func (PaginationUser) TableName() string {
	return "users"
}
