package domain

import "errors"

var (
	ErrInvalidTarget         = errors.New("invalid sync target")
	ErrInvalidDateRange      = errors.New("invalid date range")
	ErrInvalidTask           = errors.New("invalid sync task")
	ErrTaskConflict          = errors.New("sync task conflict")
	ErrTaskNotFound          = errors.New("sync task not found")
	ErrTaskNotRetryable      = errors.New("sync task is not retryable")
	ErrNoDueTask             = errors.New("no due sync task")
	ErrDataSourceUnavailable = errors.New("data source unavailable")
	ErrInvalidPagination     = errors.New("invalid pagination")
	ErrInvalidStockSymbol    = errors.New("invalid stock symbol")
	ErrStockNotFound         = errors.New("stock not found")
)
