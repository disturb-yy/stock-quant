// Package domain 定义股票池的核心业务概念和不变量。
package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MaxStockPoolNameLength        = 100
	MaxStockPoolDescriptionLength = 500
	MaxStockPoolSearchLength      = 100
)

// Source 表示股票池可追溯的创建来源。
type Source string

const (
	// SourceManual 表示由用户显式创建的股票池。
	SourceManual Source = "manual"
)

var ErrStockPoolNotFound = errors.New("stock pool not found")

// StockPool 是可独立访问的股票池身份与概览元数据。
type StockPool struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Source      Source    `json:"source"`
	MemberCount int64     `json:"member_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// StockPoolInput 是客户端可提供的最小股票池元数据。
type StockPoolInput struct {
	Name        string
	Description *string
}

// ValidationError 表示可安全暴露给 API 的字段校验失败。
type ValidationError struct {
	Fields map[string]string
}

func (err *ValidationError) Error() string {
	return "stock pool validation failed"
}

// Details 返回统一错误响应需要的字段详情。
func (err *ValidationError) Details() map[string]any {
	details := make(map[string]any, len(err.Fields))
	for field, message := range err.Fields {
		details[field] = message
	}
	return details
}

// NewManualStockPool 规范化创建输入并固定手工来源。
func NewManualStockPool(input StockPoolInput) (StockPool, error) {
	name, description, fields := normalizeInput(input)
	if len(fields) > 0 {
		return StockPool{}, &ValidationError{Fields: fields}
	}
	return StockPool{Name: name, Description: description, Source: SourceManual}, nil
}

func normalizeInput(input StockPoolInput) (string, *string, map[string]string) {
	fields := make(map[string]string)
	name := strings.TrimSpace(input.Name)
	if name == "" {
		fields["name"] = "必须提供股票池名称"
	} else if utf8.RuneCountInString(name) > MaxStockPoolNameLength {
		fields["name"] = fmt.Sprintf("长度不能超过 %d 个字符", MaxStockPoolNameLength)
	}
	description := normalizeDescription(input.Description, fields)
	return name, description, fields
}

func normalizeDescription(input *string, fields map[string]string) *string {
	if input == nil {
		return nil
	}
	description := strings.TrimSpace(*input)
	if utf8.RuneCountInString(description) > MaxStockPoolDescriptionLength {
		fields["description"] = fmt.Sprintf("长度不能超过 %d 个字符", MaxStockPoolDescriptionLength)
	}
	return &description
}
