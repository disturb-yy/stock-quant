// Package domain 定义股票分析领域模型。
package domain

import (
	"errors"
	"strings"
)

// FinancialMetric 是某只股票某个日期的财务指标快照。
type FinancialMetric struct {
	InstrumentCode string
	MetricDate     string
	MetricName     string
	Basis          string
	MetricValue    string
}

// Validate 检查 FinancialMetric 的最小领域约束。
func (metric FinancialMetric) Validate() error {
	if strings.TrimSpace(metric.InstrumentCode) == "" {
		return errors.New("financial metric instrument code is required")
	}
	if strings.TrimSpace(metric.MetricDate) == "" {
		return errors.New("financial metric date is required")
	}
	if strings.TrimSpace(metric.MetricName) == "" {
		return errors.New("financial metric name is required")
	}
	if strings.TrimSpace(metric.Basis) == "" {
		return errors.New("financial metric basis is required")
	}
	if strings.TrimSpace(metric.MetricValue) == "" {
		return errors.New("financial metric value is required")
	}
	return nil
}
