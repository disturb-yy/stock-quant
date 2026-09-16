package domain

import (
	"errors"
	"math/big"
	"sort"
	"strings"
	"time"
)

// FinancialPeriod 表示财务报告的期间口径。
type FinancialPeriod string

const (
	PeriodAnnual    FinancialPeriod = "annual"
	PeriodQuarterly FinancialPeriod = "quarterly"
)

// FinancialRange 表示查询需要返回的报告范围。
type FinancialRange string

const (
	RangeThreeYears FinancialRange = "3y"
	RangeFiveYears  FinancialRange = "5y"
)

// FinancialReport 是同一股票、期间口径和报告期的一份财务报告。
type FinancialReport struct {
	InstrumentCode string
	Period         FinancialPeriod
	PeriodEnd      string
	FiscalYear     int
	FiscalQuarter  string
	PublishedAt    *string
	Income         IncomeStatement
	Balance        BalanceSheet
	CashFlow       CashFlowStatement
}

// IncomeStatement 是简化利润表的原始金额。
type IncomeStatement struct {
	Revenue         *string
	GrossProfit     *string
	OperatingProfit *string
	NetProfit       *string
}

// BalanceSheet 是简化资产负债表的原始金额。
type BalanceSheet struct {
	CashAndEquivalents *string
	AccountsReceivable *string
	Inventory          *string
	CurrentAssets      *string
	CurrentLiabilities *string
	TotalAssets        *string
	TotalLiabilities   *string
	TotalEquity        *string
}

// CashFlowStatement 是简化现金流量表的原始金额。
type CashFlowStatement struct {
	OperatingCashFlow  *string
	CapitalExpenditure *string
	InvestingCashFlow  *string
	FinancingCashFlow  *string
	NetCashChange      *string
}

// Indicators 是由同口径报告字段计算出的研究指标。
type Indicators struct {
	RevenueYoYPct   *string
	NetProfitYoYPct *string
	GrossMarginPct  *string
	ROEPct          *string
	FreeCashFlow    *string
	DebtToAssetPct  *string
	CurrentRatio    *string
}

// Validate 检查财务报告的期间身份和最小领域约束。
func (report FinancialReport) Validate() error {
	if strings.TrimSpace(report.InstrumentCode) == "" {
		return errors.New("financial report instrument code is required")
	}
	if report.Period != PeriodAnnual && report.Period != PeriodQuarterly {
		return errors.New("financial report period is invalid")
	}
	if _, err := time.Parse("2006-01-02", report.PeriodEnd); err != nil {
		return errors.New("financial report period end is invalid")
	}
	if report.FiscalYear < 1 {
		return errors.New("financial report fiscal year is invalid")
	}
	if report.Period == PeriodAnnual && report.FiscalQuarter != "" {
		return errors.New("annual financial report cannot have fiscal quarter")
	}
	if report.Period == PeriodQuarterly && !validFiscalQuarter(report.FiscalQuarter) {
		return errors.New("quarterly financial report fiscal quarter is invalid")
	}
	return nil
}

// SelectReports 按报告期升序选择请求范围内的报告。
func SelectReports(reports []FinancialReport, period FinancialPeriod, requestedRange FinancialRange) ([]FinancialReport, error) {
	limit, years, err := rangeSpec(period, requestedRange)
	if err != nil {
		return nil, err
	}
	selected := make([]FinancialReport, 0, len(reports))
	for _, report := range reports {
		if report.Period != period {
			continue
		}
		if err := report.Validate(); err != nil {
			return nil, err
		}
		selected = append(selected, report)
	}
	sort.Slice(selected, func(left, right int) bool { return selected[left].PeriodEnd < selected[right].PeriodEnd })
	if err := rejectDuplicateReports(selected); err != nil {
		return nil, err
	}
	if len(selected) == 0 {
		return []FinancialReport{}, nil
	}
	if period == PeriodQuarterly {
		latest, _ := time.Parse("2006-01-02", selected[len(selected)-1].PeriodEnd)
		from := latest.AddDate(-years, 0, 0)
		withinRange := selected[:0]
		for _, report := range selected {
			periodEnd, _ := time.Parse("2006-01-02", report.PeriodEnd)
			if !periodEnd.Before(from) && !periodEnd.After(latest) {
				withinRange = append(withinRange, report)
			}
		}
		selected = withinRange
	}
	if len(selected) > limit {
		selected = selected[len(selected)-limit:]
	}
	return selected, nil
}

func rangeSpec(period FinancialPeriod, requestedRange FinancialRange) (int, int, error) {
	if period != PeriodAnnual && period != PeriodQuarterly {
		return 0, 0, errors.New("financial report period is invalid")
	}
	switch requestedRange {
	case RangeThreeYears:
		if period == PeriodQuarterly {
			return 12, 3, nil
		}
		return 3, 3, nil
	case RangeFiveYears:
		if period == PeriodQuarterly {
			return 20, 5, nil
		}
		return 5, 5, nil
	default:
		return 0, 0, errors.New("financial report range is invalid")
	}
}

func rejectDuplicateReports(reports []FinancialReport) error {
	seen := make(map[string]struct{}, len(reports))
	for _, report := range reports {
		key := string(report.Period) + "\x00" + report.PeriodEnd
		if _, exists := seen[key]; exists {
			return errors.New("duplicate financial report period")
		}
		seen[key] = struct{}{}
	}
	return nil
}

// CalculateIndicators 按同一期间口径计算当前报告的研究指标。
func CalculateIndicators(current FinancialReport, previous *FinancialReport) Indicators {
	if !comparableReports(current, previous) {
		previous = nil
	}
	indicators := Indicators{
		RevenueYoYPct:   yearOverYear(current.Income.Revenue, previousValue(previous, func(report FinancialReport) *string { return report.Income.Revenue })),
		NetProfitYoYPct: yearOverYear(current.Income.NetProfit, previousValue(previous, func(report FinancialReport) *string { return report.Income.NetProfit })),
		GrossMarginPct:  ratioPercent(current.Income.GrossProfit, current.Income.Revenue),
		ROEPct:          ratioPercent(current.Income.NetProfit, current.Balance.TotalEquity),
		FreeCashFlow:    freeCashFlow(current.CashFlow.OperatingCashFlow, current.CashFlow.CapitalExpenditure),
		DebtToAssetPct:  ratioPercent(current.Balance.TotalLiabilities, current.Balance.TotalAssets),
		CurrentRatio:    ratio(current.Balance.CurrentAssets, current.Balance.CurrentLiabilities),
	}
	return indicators
}

func comparableReports(current FinancialReport, previous *FinancialReport) bool {
	if previous == nil || current.Period != previous.Period || current.FiscalYear != previous.FiscalYear+1 {
		return false
	}
	return current.Period == PeriodAnnual || current.FiscalQuarter == previous.FiscalQuarter
}

func previousValue(previous *FinancialReport, value func(FinancialReport) *string) *string {
	if previous == nil {
		return nil
	}
	return value(*previous)
}

func yearOverYear(current, previous *string) *string {
	if current == nil || previous == nil {
		return nil
	}
	currentValue, ok := rationalValue(current)
	if !ok {
		return nil
	}
	previousValue, ok := rationalValue(previous)
	if !ok || previousValue.Sign() == 0 {
		return nil
	}
	difference := new(big.Rat).Sub(currentValue, previousValue)
	return formatRat(new(big.Rat).Quo(difference, previousValue), true)
}

func freeCashFlow(operatingCashFlow, capitalExpenditure *string) *string {
	if operatingCashFlow == nil || capitalExpenditure == nil {
		return nil
	}
	capitalExpenditure = CanonicalCapitalExpenditure(capitalExpenditure)
	if capitalExpenditure == nil {
		return nil
	}
	operating, ok := rationalValue(operatingCashFlow)
	if !ok {
		return nil
	}
	capital, ok := rationalValue(capitalExpenditure)
	if !ok {
		return nil
	}
	return formatRat(new(big.Rat).Sub(operating, capital), false)
}

// CanonicalCapitalExpenditure 只保留可确认的非负资本开支现金流出额。
func CanonicalCapitalExpenditure(value *string) *string {
	if value == nil {
		return nil
	}
	parsed, ok := rationalValue(value)
	if !ok || parsed.Sign() < 0 {
		return nil
	}
	return value
}

func ratioPercent(numerator, denominator *string) *string {
	return calculateRatio(numerator, denominator, true)
}

func ratio(numerator, denominator *string) *string {
	return calculateRatio(numerator, denominator, false)
}

func calculateRatio(numerator, denominator *string, percent bool) *string {
	if numerator == nil || denominator == nil {
		return nil
	}
	numeratorValue, ok := rationalValue(numerator)
	if !ok {
		return nil
	}
	denominatorValue, ok := rationalValue(denominator)
	if !ok || denominatorValue.Sign() == 0 {
		return nil
	}
	result := new(big.Rat).Quo(numeratorValue, denominatorValue)
	return formatRat(result, percent)
}

func rationalValue(value *string) (*big.Rat, bool) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, false
	}
	rational, ok := new(big.Rat).SetString(strings.TrimSpace(*value))
	return rational, ok
}

func formatRat(value *big.Rat, percent bool) *string {
	if percent {
		value = new(big.Rat).Mul(value, big.NewRat(100, 1))
	}
	formatted := value.FloatString(2)
	return &formatted
}

func validFiscalQuarter(value string) bool {
	return value == "Q1" || value == "Q2" || value == "Q3" || value == "Q4"
}
