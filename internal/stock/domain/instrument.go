// Package domain 定义股票基础信息领域模型。
package domain

import (
	"errors"
	"strings"
)

// InstrumentStatus 表示股票当前的上市状态。
type InstrumentStatus string

const (
	// InstrumentStatusActive 表示可用于演示数据查询的股票。
	InstrumentStatusActive InstrumentStatus = "active"
)

// Instrument 是股票基础信息实体。
type Instrument struct {
	Code     string
	Name     string
	Exchange string
	Status   InstrumentStatus
	AsOf     string
}

// Validate 检查 Instrument 的最小领域约束。
func (instrument Instrument) Validate() error {
	if strings.TrimSpace(instrument.Code) == "" {
		return errors.New("instrument code is required")
	}
	if strings.TrimSpace(instrument.Name) == "" {
		return errors.New("instrument name is required")
	}
	if strings.TrimSpace(instrument.Exchange) == "" {
		return errors.New("instrument exchange is required")
	}
	if instrument.Status == "" {
		return errors.New("instrument status is required")
	}
	if strings.TrimSpace(instrument.AsOf) == "" {
		return errors.New("instrument as-of date is required")
	}
	return nil
}
