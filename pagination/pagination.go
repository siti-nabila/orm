package pagination

import (
	"math"

	"github.com/siti-nabila/orm/pkg/dictionary"
)

const (
	DefaultLimit = 10
	MaxLimit     = 1000
	LimitMax     = -1
)

type (
	Config struct {
		DefaultLimit int
		MaxLimit     int
	}

	Params struct {
		Page  int
		Limit int
	}

	PaginationOptions struct {
		Page     int
		PerPage  int
		MaxLimit int
	}

	Meta struct {
		Page       int
		Limit      int
		HasNext    bool
		HasPrev    bool
		TotalItems *int64
		TotalPages *int
	}

	Result[T any] struct {
		Data []T
		Meta Meta
	}

	PageInfo struct {
		Page       int
		PerPage    int
		TotalRows  int64
		TotalPages int
		HasNext    bool
		HasPrev    bool
	}
)

func Normalize(params Params) (Params, error) {
	return NormalizeWithConfig(params, Config{})
}

func NormalizeWithConfig(params Params, cfg Config) (Params, error) {
	cfg = normalizeConfig(cfg)

	if params.Page < 1 {
		params.Page = 1
	}

	switch {
	case params.Limit == 0:
		params.Limit = cfg.DefaultLimit
	case params.Limit == LimitMax:
		params.Limit = cfg.MaxLimit
	case params.Limit < LimitMax:
		return Params{}, dictionary.ErrPaginationInvalidLimit
	case params.Limit > cfg.MaxLimit:
		params.Limit = cfg.MaxLimit
	}

	if params.Page > 1 && params.Page-1 > math.MaxInt/params.Limit {
		return Params{}, dictionary.ErrPaginationOffsetOverflow
	}
	return params, nil
}

func NormalizeOptionsWithConfig(opts PaginationOptions, cfg Config) (PaginationOptions, error) {
	if opts.PerPage < 0 {
		return PaginationOptions{}, dictionary.ErrPaginationInvalidLimit
	}

	if opts.MaxLimit > 0 {
		cfg.MaxLimit = opts.MaxLimit
	}

	params, err := NormalizeWithConfig(Params{
		Page:  opts.Page,
		Limit: opts.PerPage,
	}, cfg)
	if err != nil {
		return PaginationOptions{}, err
	}

	return PaginationOptions{
		Page:     params.Page,
		PerPage:  params.Limit,
		MaxLimit: opts.MaxLimit,
	}, nil
}

func normalizeConfig(cfg Config) Config {
	if cfg.MaxLimit < 1 {
		cfg.MaxLimit = MaxLimit
	}
	if cfg.DefaultLimit < 1 {
		cfg.DefaultLimit = DefaultLimit
	}
	if cfg.DefaultLimit > cfg.MaxLimit {
		cfg.DefaultLimit = cfg.MaxLimit
	}
	return cfg
}

func Offset(params Params) int {
	return (params.Page - 1) * params.Limit
}

func OptionsOffset(opts PaginationOptions) int {
	return (opts.Page - 1) * opts.PerPage
}

func BuildPageInfo(opts PaginationOptions, totalRows int64) PageInfo {
	totalPages := calculateTotalPages(totalRows, opts.PerPage)
	return PageInfo{
		Page:       opts.Page,
		PerPage:    opts.PerPage,
		TotalRows:  totalRows,
		TotalPages: totalPages,
		HasNext:    opts.Page < totalPages,
		HasPrev:    opts.Page > 1 && totalPages > 0,
	}
}

func calculateTotalPages(totalRows int64, perPage int) int {
	if totalRows <= 0 || perPage <= 0 {
		return 0
	}

	return int((totalRows + int64(perPage) - 1) / int64(perPage))
}

func BuildMeta(params Params, itemCount int, hasNext bool, total *int64) Meta {
	meta := Meta{
		Page:       params.Page,
		Limit:      params.Limit,
		HasNext:    hasNext,
		HasPrev:    params.Page > 1,
		TotalItems: total,
	}

	if total != nil {
		totalPages := 0
		if *total > 0 {
			totalPages = int((*total + int64(params.Limit) - 1) / int64(params.Limit))
		}
		meta.TotalPages = &totalPages
		meta.HasNext = int64(Offset(params)+itemCount) < *total
	}

	return meta
}
