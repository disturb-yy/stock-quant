package stock

import (
	"context"
	"errors"
	"fmt"
	"strings"

	stockdomain "github.com/disturb-yy/stock-quant/internal/stock/domain"
)

const (
	defaultFinancialPeriod   = stockdomain.PeriodAnnual
	defaultFinancialRange    = stockdomain.RangeFiveYears
	FinancialPeriodAnnual    = stockdomain.PeriodAnnual
	FinancialPeriodQuarterly = stockdomain.PeriodQuarterly
	FinancialRangeThreeYears = stockdomain.RangeThreeYears
	FinancialRangeFiveYears  = stockdomain.RangeFiveYears
)

var (
	// ErrFinancialsInstrumentNotFound 表示财务查询的股票身份不存在。
	ErrFinancialsInstrumentNotFound = errors.New("financials instrument not found")
)

// FinancialsRequest 是股票财务报告查询的应用层请求。
type FinancialsRequest struct {
	Symbol string
	Period stockdomain.FinancialPeriod
	Range  stockdomain.FinancialRange
}

// FinancialsSnapshot 是基础设施返回的财务报告和来源元数据。
type FinancialsSnapshot struct {
	Symbol      string
	Name        string
	SeedVersion string
	AsOf        string
	Reports     []stockdomain.FinancialReport
}

// FinancialsReader 是股票财务报告查询所需的最小数据访问边界。
type FinancialsReader interface {
	ReadStockFinancials(context.Context, FinancialsRequest) (FinancialsSnapshot, error)
}

// FinancialSource 是财务数据来源模式，具体 seed 版本由查询结果返回。
type FinancialSource struct {
	Mode     string
	Provider string
}

// FinancialsValidationError 表示股票财务查询参数不符合契约。
type FinancialsValidationError struct {
	Fields map[string]string
}

func (err *FinancialsValidationError) Error() string {
	return "stock financials request parameters are invalid"
}

// Details 返回可安全暴露的参数诊断字段。
func (err *FinancialsValidationError) Details() map[string]any {
	fields := make(map[string]any, len(err.Fields))
	for field, message := range err.Fields {
		fields[field] = message
	}
	return map[string]any{"fields": fields}
}

// FinancialsService 编排股票财务报告读取和指标计算。
type FinancialsService struct {
	reader FinancialsReader
	source FinancialSource
}

// NewFinancialsService 创建股票财务报告查询服务。
func NewFinancialsService(reader FinancialsReader, source FinancialSource) (*FinancialsService, error) {
	if reader == nil {
		return nil, errors.New("stock financials reader is required")
	}
	if strings.TrimSpace(source.Mode) == "" || strings.TrimSpace(source.Provider) == "" {
		return nil, errors.New("stock financials source is required")
	}
	return &FinancialsService{reader: reader, source: source}, nil
}

// Financials 读取同一报告期口径下的财务摘要、趋势和简化报表。
func (service *FinancialsService) Financials(ctx context.Context, request FinancialsRequest) (StockFinancials, error) {
	normalized, err := normalizeFinancialsRequest(request)
	if err != nil {
		return StockFinancials{}, err
	}
	snapshot, err := service.reader.ReadStockFinancials(ctx, normalized)
	if err != nil {
		return StockFinancials{}, fmt.Errorf("read stock financials %q: %w", normalized.Symbol, err)
	}
	if err := validateFinancialsSnapshot(snapshot, normalized.Symbol); err != nil {
		return StockFinancials{}, err
	}
	reports, err := stockdomain.SelectReports(snapshot.Reports, normalized.Period, normalized.Range)
	if err != nil {
		return StockFinancials{}, fmt.Errorf("select stock financial reports %q: %w", normalized.Symbol, err)
	}
	return buildStockFinancials(snapshot, normalized, reports, service.source), nil
}

func normalizeFinancialsRequest(request FinancialsRequest) (FinancialsRequest, error) {
	fields := make(map[string]string)
	if strings.TrimSpace(request.Symbol) != request.Symbol || !symbolPattern.MatchString(request.Symbol) {
		fields["symbol"] = "必须是 Markets 返回的股票代码"
	}
	period := request.Period
	if period == "" {
		period = defaultFinancialPeriod
	} else if period != stockdomain.PeriodAnnual && period != stockdomain.PeriodQuarterly {
		fields["period"] = "必须是 annual 或 quarterly"
	}
	requestedRange := request.Range
	if requestedRange == "" {
		requestedRange = defaultFinancialRange
	} else if requestedRange != stockdomain.RangeThreeYears && requestedRange != stockdomain.RangeFiveYears {
		fields["range"] = "必须是 3y 或 5y"
	}
	if len(fields) > 0 {
		return FinancialsRequest{}, &FinancialsValidationError{Fields: fields}
	}
	return FinancialsRequest{Symbol: request.Symbol, Period: period, Range: requestedRange}, nil
}

func validateFinancialsSnapshot(snapshot FinancialsSnapshot, symbol string) error {
	if snapshot.Symbol != symbol {
		return fmt.Errorf("stock financials identity mismatch: requested %q, got %q", symbol, snapshot.Symbol)
	}
	if strings.TrimSpace(snapshot.Name) == "" || strings.TrimSpace(snapshot.SeedVersion) == "" || strings.TrimSpace(snapshot.AsOf) == "" {
		return errors.New("stock financials name, seed version and as-of are required")
	}
	for _, report := range snapshot.Reports {
		if report.InstrumentCode != symbol {
			return fmt.Errorf("stock financial report identity mismatch: requested %q, got %q", symbol, report.InstrumentCode)
		}
	}
	return nil
}

// StockFinancials 是股票财务摘要、趋势和简化报表 API 响应。
type StockFinancials struct {
	Symbol            string                      `json:"symbol"`
	Name              string                      `json:"name"`
	Period            stockdomain.FinancialPeriod `json:"period"`
	RequestedRange    stockdomain.FinancialRange  `json:"requested_range"`
	EffectiveRange    FinancialEffectiveRange     `json:"effective_range"`
	ReportingCurrency string                      `json:"reporting_currency"`
	AmountUnit        string                      `json:"amount_unit"`
	LatestReportDate  *string                     `json:"latest_report_date"`
	Summary           *FinancialSummary           `json:"summary"`
	Reports           []FinancialReportView       `json:"reports"`
	Source            FinancialDataSource         `json:"source"`
}

// FinancialEffectiveRange 是实际返回报告期的首尾范围。
type FinancialEffectiveRange struct {
	From *string `json:"from"`
	To   *string `json:"to"`
}

// FinancialSummary 是最新有效报告期的摘要。
type FinancialSummary struct {
	PeriodEnd         string  `json:"period_end"`
	PublishedAt       *string `json:"published_at"`
	Revenue           *string `json:"revenue"`
	RevenueYoYPct     *string `json:"revenue_yoy_pct"`
	NetProfit         *string `json:"net_profit"`
	NetProfitYoYPct   *string `json:"net_profit_yoy_pct"`
	GrossMarginPct    *string `json:"gross_margin_pct"`
	ROEPct            *string `json:"roe_pct"`
	OperatingCashFlow *string `json:"operating_cash_flow"`
	FreeCashFlow      *string `json:"free_cash_flow"`
	DebtToAssetPct    *string `json:"debt_to_asset_pct"`
	CurrentRatio      *string `json:"current_ratio"`
}

// FinancialReportView 是一份简化财务报告及其服务端指标。
type FinancialReportView struct {
	PeriodEnd     string              `json:"period_end"`
	FiscalYear    int                 `json:"fiscal_year"`
	FiscalQuarter *string             `json:"fiscal_quarter"`
	PublishedAt   *string             `json:"published_at"`
	Income        FinancialIncome     `json:"income"`
	Balance       FinancialBalance    `json:"balance"`
	CashFlow      FinancialCashFlow   `json:"cash_flow"`
	Indicators    FinancialIndicators `json:"indicators"`
}

// FinancialIncome 是简化利润表。
type FinancialIncome struct {
	Revenue         *string `json:"revenue"`
	GrossProfit     *string `json:"gross_profit"`
	OperatingProfit *string `json:"operating_profit"`
	NetProfit       *string `json:"net_profit"`
}

// FinancialBalance 是简化资产负债表。
type FinancialBalance struct {
	CashAndEquivalents *string `json:"cash_and_equivalents"`
	AccountsReceivable *string `json:"accounts_receivable"`
	Inventory          *string `json:"inventory"`
	CurrentAssets      *string `json:"current_assets"`
	CurrentLiabilities *string `json:"current_liabilities"`
	TotalAssets        *string `json:"total_assets"`
	TotalLiabilities   *string `json:"total_liabilities"`
	TotalEquity        *string `json:"total_equity"`
}

// FinancialCashFlow 是简化现金流量表。
type FinancialCashFlow struct {
	OperatingCashFlow  *string `json:"operating_cash_flow"`
	CapitalExpenditure *string `json:"capital_expenditure"`
	InvestingCashFlow  *string `json:"investing_cash_flow"`
	FinancingCashFlow  *string `json:"financing_cash_flow"`
	NetCashChange      *string `json:"net_cash_change"`
}

// FinancialIndicators 是服务端计算的同比、比率和自由现金流。
type FinancialIndicators struct {
	RevenueYoYPct   *string `json:"revenue_yoy_pct"`
	NetProfitYoYPct *string `json:"net_profit_yoy_pct"`
	GrossMarginPct  *string `json:"gross_margin_pct"`
	ROEPct          *string `json:"roe_pct"`
	FreeCashFlow    *string `json:"free_cash_flow"`
	DebtToAssetPct  *string `json:"debt_to_asset_pct"`
	CurrentRatio    *string `json:"current_ratio"`
}

// FinancialDataSource 描述财务数据的来源、版本和截至日期。
type FinancialDataSource struct {
	Mode        string `json:"mode"`
	Provider    string `json:"provider"`
	SeedVersion string `json:"seed_version"`
	AsOf        string `json:"as_of"`
}

func buildStockFinancials(snapshot FinancialsSnapshot, request FinancialsRequest, reports []stockdomain.FinancialReport, source FinancialSource) StockFinancials {
	result := StockFinancials{
		Symbol: snapshot.Symbol, Name: snapshot.Name, Period: request.Period, RequestedRange: request.Range,
		ReportingCurrency: "CNY", AmountUnit: "CNY", Reports: make([]FinancialReportView, 0, len(reports)),
		Source: FinancialDataSource{Mode: source.Mode, Provider: source.Provider, SeedVersion: snapshot.SeedVersion, AsOf: snapshot.AsOf},
	}
	if len(reports) == 0 {
		return result
	}
	from, to := reports[0].PeriodEnd, reports[len(reports)-1].PeriodEnd
	result.EffectiveRange = FinancialEffectiveRange{From: &from, To: &to}
	latest := reports[len(reports)-1]
	result.LatestReportDate = &latest.PeriodEnd
	previous := previousFinancialReports(snapshot.Reports, request.Period)
	for _, report := range reports {
		indicators := stockdomain.CalculateIndicators(report, previousReport(previous, report))
		result.Reports = append(result.Reports, financialReportView(report, indicators))
	}
	latestIndicators := stockdomain.CalculateIndicators(latest, previousReport(previous, latest))
	result.Summary = financialSummary(latest, latestIndicators)
	return result
}

func previousFinancialReports(reports []stockdomain.FinancialReport, period stockdomain.FinancialPeriod) map[string]stockdomain.FinancialReport {
	previous := make(map[string]stockdomain.FinancialReport)
	for _, report := range reports {
		if report.Period == period {
			previous[financialReportKey(report)] = report
		}
	}
	return previous
}

func financialReportKey(report stockdomain.FinancialReport) string {
	if report.Period == stockdomain.PeriodAnnual {
		return fmt.Sprintf("%s:%d", report.Period, report.FiscalYear)
	}
	return fmt.Sprintf("%s:%d:%s", report.Period, report.FiscalYear, report.FiscalQuarter)
}

func previousReportKey(report stockdomain.FinancialReport) string {
	if report.Period == stockdomain.PeriodAnnual {
		return fmt.Sprintf("%s:%d", report.Period, report.FiscalYear-1)
	}
	return fmt.Sprintf("%s:%d:%s", report.Period, report.FiscalYear-1, report.FiscalQuarter)
}

func previousReport(reports map[string]stockdomain.FinancialReport, current stockdomain.FinancialReport) *stockdomain.FinancialReport {
	report, exists := reports[previousReportKey(current)]
	if !exists || report.PeriodEnd == current.PeriodEnd {
		return nil
	}
	return &report
}

func financialReportView(report stockdomain.FinancialReport, indicators stockdomain.Indicators) FinancialReportView {
	return FinancialReportView{
		PeriodEnd: report.PeriodEnd, FiscalYear: report.FiscalYear, FiscalQuarter: optionalFinancialString(report.FiscalQuarter), PublishedAt: report.PublishedAt,
		Income:     FinancialIncome{Revenue: report.Income.Revenue, GrossProfit: report.Income.GrossProfit, OperatingProfit: report.Income.OperatingProfit, NetProfit: report.Income.NetProfit},
		Balance:    FinancialBalance{CashAndEquivalents: report.Balance.CashAndEquivalents, AccountsReceivable: report.Balance.AccountsReceivable, Inventory: report.Balance.Inventory, CurrentAssets: report.Balance.CurrentAssets, CurrentLiabilities: report.Balance.CurrentLiabilities, TotalAssets: report.Balance.TotalAssets, TotalLiabilities: report.Balance.TotalLiabilities, TotalEquity: report.Balance.TotalEquity},
		CashFlow:   FinancialCashFlow{OperatingCashFlow: report.CashFlow.OperatingCashFlow, CapitalExpenditure: stockdomain.CanonicalCapitalExpenditure(report.CashFlow.CapitalExpenditure), InvestingCashFlow: report.CashFlow.InvestingCashFlow, FinancingCashFlow: report.CashFlow.FinancingCashFlow, NetCashChange: report.CashFlow.NetCashChange},
		Indicators: FinancialIndicators{RevenueYoYPct: indicators.RevenueYoYPct, NetProfitYoYPct: indicators.NetProfitYoYPct, GrossMarginPct: indicators.GrossMarginPct, ROEPct: indicators.ROEPct, FreeCashFlow: indicators.FreeCashFlow, DebtToAssetPct: indicators.DebtToAssetPct, CurrentRatio: indicators.CurrentRatio},
	}
}

func financialSummary(report stockdomain.FinancialReport, indicators stockdomain.Indicators) *FinancialSummary {
	return &FinancialSummary{PeriodEnd: report.PeriodEnd, PublishedAt: report.PublishedAt, Revenue: report.Income.Revenue, RevenueYoYPct: indicators.RevenueYoYPct, NetProfit: report.Income.NetProfit, NetProfitYoYPct: indicators.NetProfitYoYPct, GrossMarginPct: indicators.GrossMarginPct, ROEPct: indicators.ROEPct, OperatingCashFlow: report.CashFlow.OperatingCashFlow, FreeCashFlow: indicators.FreeCashFlow, DebtToAssetPct: indicators.DebtToAssetPct, CurrentRatio: indicators.CurrentRatio}
}

func optionalFinancialString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
