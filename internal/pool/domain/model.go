// Package domain 定义股票池的核心业务概念和不变量。
package domain

import (
	"errors"
	"fmt"
	"regexp"
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
	// SourceScreener 仅表示 SCR-003 已真实持久化的筛选来源。
	SourceScreener Source = "screener"
)

var ErrStockPoolNotFound = errors.New("stock pool not found")

var (
	// ErrStockPoolSourceMetadataUnavailable 表示来源字段缺少可追溯的真实元数据。
	ErrStockPoolSourceMetadataUnavailable = errors.New("stock pool source metadata unavailable")
	// ErrStockPoolMemberNotFound 表示指定成员不在股票池中。
	ErrStockPoolMemberNotFound = errors.New("stock pool member not found")
	// ErrStockPoolMemberConflict 表示成员已存在于股票池中。
	ErrStockPoolMemberConflict = errors.New("stock pool member already exists")
	// ErrStockPoolInstrumentNotFound 表示股票身份不存在。
	ErrStockPoolInstrumentNotFound = errors.New("stock pool instrument not found")
	stockSymbolPattern             = regexp.MustCompile(`^[A-Za-z0-9]{1,16}\.[A-Za-z]{2,8}$`)
)

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

// StockPoolMember 是股票池成员的真实股票身份和展示名称。
type StockPoolMember struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}

// NewStockPoolMember 校验并保留 Markets 返回的原始股票代码。
func NewStockPoolMember(symbol string) (StockPoolMember, error) {
	if strings.TrimSpace(symbol) != symbol || !stockSymbolPattern.MatchString(symbol) {
		return StockPoolMember{}, &ValidationError{Fields: map[string]string{"symbol": "必须是 Markets 返回的股票代码"}}
	}
	return StockPoolMember{Symbol: symbol}, nil
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
