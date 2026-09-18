package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MaxSavedScreenerNameLength        = 100
	MaxSavedScreenerDescriptionLength = 500
)

// SavedScreener 是可复用的选股方案身份和当前规范化版本。
type SavedScreener struct {
	ID          int64        `json:"id"`
	Name        string       `json:"name"`
	Description *string      `json:"description"`
	Spec        ScreenerSpec `json:"spec"`
	Version     int64        `json:"version"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// SavedScreenerInput 是创建或更新方案时的规范化输入。
type SavedScreenerInput struct {
	Name        string
	Description *string
	Spec        ScreenerSpec
}

var (
	ErrSavedScreenerNotFound     = errors.New("saved screener not found")
	ErrSavedScreenerVersionStale = errors.New("saved screener version is stale")
)

// SavedScreenerVersionConflictError 表示更新使用了过期的版本号。
type SavedScreenerVersionConflictError struct {
	CurrentVersion int64
}

func (err *SavedScreenerVersionConflictError) Error() string {
	return fmt.Sprintf("saved screener version conflict, current version is %d", err.CurrentVersion)
}

func (err *SavedScreenerVersionConflictError) Unwrap() error {
	return ErrSavedScreenerVersionStale
}

func (err *SavedScreenerVersionConflictError) Details() map[string]any {
	return map[string]any{"current_version": err.CurrentVersion}
}

// NormalizeSavedScreenerInput 统一校验方案字段并复用 SCR-001 的 spec 规范化。
func NormalizeSavedScreenerInput(input SavedScreenerInput) (SavedScreenerInput, error) {
	normalized := input
	normalized.Name = strings.TrimSpace(input.Name)
	fields := make(map[string]string)
	if normalized.Name == "" {
		fields["name"] = "必须提供方案名称"
	} else if utf8.RuneCountInString(normalized.Name) > MaxSavedScreenerNameLength {
		fields["name"] = fmt.Sprintf("长度不能超过 %d 个字符", MaxSavedScreenerNameLength)
	}
	if input.Description != nil {
		description := strings.TrimSpace(*input.Description)
		normalized.Description = &description
		if utf8.RuneCountInString(description) > MaxSavedScreenerDescriptionLength {
			fields["description"] = fmt.Sprintf("长度不能超过 %d 个字符", MaxSavedScreenerDescriptionLength)
		}
	}

	normalizedSpec, err := NormalizeAndValidateSpec(input.Spec)
	if err != nil {
		appendSavedSpecValidation(fields, err)
	} else {
		normalized.Spec = normalizedSpec
	}
	if len(fields) > 0 {
		return SavedScreenerInput{}, &ValidationError{Fields: fields}
	}
	return normalized, nil
}

func appendSavedSpecValidation(fields map[string]string, err error) {
	var validationError *ValidationError
	if errors.As(err, &validationError) {
		for field, message := range validationError.Fields {
			fields["spec."+field] = message
		}
		return
	}
	if errors.Is(err, ErrUniverseNotFound) {
		fields["spec.universe_id"] = "选股 Universe 不存在"
		return
	}
	fields["spec"] = "选股规格无效"
}
