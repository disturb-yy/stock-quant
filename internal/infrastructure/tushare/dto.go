package tushare

import "stock-quant/internal/shared/types"

// StockBasicDTO 是 stock_basic 的 provider 字段模型。
type StockBasicDTO struct {
	TSCode      string
	Symbol      string
	Name        string
	Area        string
	Industry    string
	FullName    *string
	EnglishName *string
	CNSPELL     string
	Market      string
	Exchange    *string
	Currency    *string
	ListStatus  *string
	ListDate    string
	DelistDate  *string
	IsHS        *string
	ActName     *string
	ActEntType  *string
}

// TradeCalendarDTO 是 trade_cal 的 provider 字段模型。
type TradeCalendarDTO struct {
	Exchange     string
	CalDate      string
	IsOpen       string
	PretradeDate *string
}

// DailyDTO 是 daily 的 provider 字段模型，金额和数量仍保持 provider 单位。
type DailyDTO struct {
	TSCode    string
	TradeDate string
	Open      types.Decimal
	High      types.Decimal
	Low       types.Decimal
	Close     types.Decimal
	PreClose  *types.Decimal
	Change    *types.Decimal
	PctChg    *types.Decimal
	Volume    types.Decimal
	Amount    types.Decimal
	AHVolume  *types.Decimal
	AHAmount  *types.Decimal
}

// AdjFactorDTO 是 adj_factor 的 provider 字段模型。
type AdjFactorDTO struct {
	TSCode    string
	TradeDate string
	AdjFactor types.Decimal
}

// SuspendDDTO 保留 suspend_d 的事件语义，不推导每日停牌状态。
type SuspendDDTO struct {
	TSCode        string
	TradeDate     string
	SuspendTiming *string
	SuspendType   string
}

// StkLimitDTO 是 stk_limit 的 provider 字段模型。
type StkLimitDTO struct {
	TradeDate string
	TSCode    string
	PreClose  *types.Decimal
	UpLimit   types.Decimal
	DownLimit types.Decimal
	AssetType *string
	Exchange  *string
}
