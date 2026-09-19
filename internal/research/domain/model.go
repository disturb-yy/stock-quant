// Package domain 定义 Research 项目的核心业务概念和不变量。
package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MaxResearchNameLength        = 100
	MaxResearchDescriptionLength = 500
)

// ResearchProject 是可独立访问的 Research 项目基础元数据。
type ResearchProject struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ResearchProjectInput 是客户端可以提供的最小项目元数据。
type ResearchProjectInput struct {
	Name        string
	Description *string
}

var ErrResearchProjectNotFound = errors.New("research project not found")

// ValidationError 表示可安全暴露给 API 的字段校验失败。
type ValidationError struct {
	Fields map[string]string
}

func (err *ValidationError) Error() string {
	return "research project validation failed"
}

// Details 返回统一错误响应需要的字段详情。
func (err *ValidationError) Details() map[string]any {
	details := make(map[string]any, len(err.Fields))
	for field, message := range err.Fields {
		details[field] = message
	}
	return details
}

// NormalizeResearchProjectInput 规范化并校验项目名称和可选描述。
func NormalizeResearchProjectInput(input ResearchProjectInput) (ResearchProjectInput, error) {
	normalized := input
	normalized.Name = strings.TrimSpace(input.Name)
	fields := make(map[string]string)
	if normalized.Name == "" {
		fields["name"] = "必须提供 Research 项目名称"
	} else if utf8.RuneCountInString(normalized.Name) > MaxResearchNameLength {
		fields["name"] = fmt.Sprintf("长度不能超过 %d 个字符", MaxResearchNameLength)
	}
	if input.Description != nil {
		description := strings.TrimSpace(*input.Description)
		normalized.Description = &description
		if utf8.RuneCountInString(description) > MaxResearchDescriptionLength {
			fields["description"] = fmt.Sprintf("长度不能超过 %d 个字符", MaxResearchDescriptionLength)
		}
	}
	if len(fields) > 0 {
		return ResearchProjectInput{}, &ValidationError{Fields: fields}
	}
	return normalized, nil
}
