package domain

import "time"

// SummaryAvailability 表示一个画像摘要的真实可用状态。
type SummaryAvailability string

const (
	SummaryAvailable   SummaryAvailability = "available"
	SummaryEmpty       SummaryAvailability = "empty"
	SummaryUnavailable SummaryAvailability = "unavailable"
)

// StockPoolSourceSummary 是股票池来源事实，不接受客户端推断或覆盖。
type StockPoolSourceSummary struct {
	Type      Source    `json:"type"`
	Reference *string   `json:"reference"`
	CreatedAt time.Time `json:"created_at"`
}

// StockPoolIndustryBucket 是一个行业的真实成员分布。
type StockPoolIndustryBucket struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	MemberCount int64  `json:"member_count"`
}

// StockPoolIndustrySummary 是行业分布及其数据日期/来源语义。
type StockPoolIndustrySummary struct {
	Availability      SummaryAvailability       `json:"availability"`
	Distribution      []StockPoolIndustryBucket `json:"distribution"`
	AsOf              *string                   `json:"as_of"`
	Provenance        *string                   `json:"provenance"`
	UnavailableReason *string                   `json:"unavailable_reason"`
}

// StockPoolMetricSummary 是 PE/ROE 的简单聚合及数据来源语义。
type StockPoolMetricSummary struct {
	Availability      SummaryAvailability `json:"availability"`
	Value             *string             `json:"value"`
	SampleSize        int64               `json:"sample_size"`
	AsOf              *string             `json:"as_of"`
	Basis             *string             `json:"basis"`
	Provenance        *string             `json:"provenance"`
	UnavailableReason *string             `json:"unavailable_reason"`
}

// StockPoolSummary 是股票池详情与基础画像的同一份一致性快照。
type StockPoolSummary struct {
	ID          int64                    `json:"id"`
	Name        string                   `json:"name"`
	Description *string                  `json:"description"`
	Source      StockPoolSourceSummary   `json:"source"`
	MemberCount int64                    `json:"member_count"`
	CreatedAt   time.Time                `json:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at"`
	Industry    StockPoolIndustrySummary `json:"industry"`
	PE          StockPoolMetricSummary   `json:"pe"`
	ROE         StockPoolMetricSummary   `json:"roe"`
}
