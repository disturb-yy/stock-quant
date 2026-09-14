package api

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// PaginationValidationError 表示分页查询参数不符合约定。
type PaginationValidationError struct {
	Fields map[string]string
}

func (e *PaginationValidationError) Error() string {
	return "pagination parameters are invalid"
}

// Details 返回可以直接放入 ErrorResponse 的安全字段信息。
func (e *PaginationValidationError) Details() map[string]any {
	fields := make(map[string]any, len(e.Fields))
	for field, message := range e.Fields {
		fields[field] = message
	}
	return map[string]any{"fields": fields}
}

// PaginationRequest 是列表接口通用的分页请求参数。
type PaginationRequest struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// DefaultPagination 返回分页参数的默认值。
func DefaultPagination() PaginationRequest {
	return PaginationRequest{Page: DefaultPage, PageSize: DefaultPageSize}
}

// ParsePagination 解析并校验 page、page_size 查询参数。
// 缺少参数使用默认值；出现重复、非整数、越界值时返回结构化校验错误。
func ParsePagination(values url.Values) (PaginationRequest, error) {
	pagination := DefaultPagination()
	fieldErrors := make(map[string]string)

	page, message := parsePaginationInteger(values, "page", DefaultPage)
	if message != "" {
		fieldErrors["page"] = message
	} else {
		pagination.Page = page
	}
	if pagination.Page < 1 && message == "" {
		fieldErrors["page"] = "必须是大于等于 1 的整数"
	}

	pageSize, message := parsePaginationInteger(values, "page_size", DefaultPageSize)
	if message != "" {
		fieldErrors["page_size"] = message
	} else {
		pagination.PageSize = pageSize
	}
	if (pagination.PageSize < 1 || pagination.PageSize > MaxPageSize) && message == "" {
		fieldErrors["page_size"] = fmt.Sprintf("必须是 1 到 %d 之间的整数", MaxPageSize)
	}

	if len(fieldErrors) > 0 {
		return pagination, &PaginationValidationError{Fields: fieldErrors}
	}

	return pagination, nil
}

func parsePaginationInteger(values url.Values, name string, defaultValue int) (int, string) {
	rawValues, ok := values[name]
	if !ok {
		return defaultValue, ""
	}
	if len(rawValues) != 1 || strings.TrimSpace(rawValues[0]) == "" {
		return 0, "必须是整数且只能提供一次"
	}

	value, err := strconv.Atoi(rawValues[0])
	if err != nil {
		return 0, "必须是整数"
	}
	return value, ""
}

// PaginationMeta 是列表响应中统一的分页元数据。
type PaginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"total_pages"`
}
