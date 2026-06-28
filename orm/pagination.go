package orm

import "github.com/siti-nabila/orm/pagination"

type (
	PaginationOptions = pagination.PaginationOptions
	PageMeta          = pagination.PageMeta
	PageData[T any]   = pagination.PageData[T]
	OffsetMode        = pagination.OffsetMode
)

const (
	OffsetModeQuery    = pagination.OffsetModeQuery
	OffsetModeInMemory = pagination.OffsetModeInMemory
)

func NewPageData[T any](items []T, meta PageMeta) PageData[T] {
	return pagination.NewPageData(items, meta)
}

func EmptyPageData[T any](page, limit int) PageData[T] {
	return pagination.EmptyPageData[T](page, limit)
}
