package demo

import (
	"fmt"
	"strconv"

	stockdomain "github.com/disturb-yy/stock-quant/internal/stock/domain"
)

type financialSeedValue struct {
	PeriodEnd     string
	FiscalYear    int
	FiscalQuarter string
	Revenue       float64
	GrossMargin   float64
	NetMargin     float64
	CurrentAssets float64
	CurrentLiab   float64
	TotalAssets   float64
	DebtRatio     float64
	OperatingCash float64
	CapitalExpend *float64
	MissingInv    bool
}

// financialReportFixture 返回带明确版本来源的财务报告演示样本。
// 数值是可重复的本地 demo 数据，不代表实时披露；缺失字段保持为 NULL 语义。
func financialReportFixture() []stockdomain.FinancialReport {
	values := annualFinancialSeedValues()
	values = append(values, quarterlyFinancialSeedValues()...)
	reports := make([]stockdomain.FinancialReport, 0, len(values))
	for _, value := range values {
		reports = append(reports, buildFinancialReport(value))
	}
	return reports
}

func annualFinancialSeedValues() []financialSeedValue {
	return []financialSeedValue{
		{PeriodEnd: "2018-12-31", FiscalYear: 2018, Revenue: 900, GrossMargin: 0.38, NetMargin: 0.15, CurrentAssets: 520, CurrentLiab: 250, TotalAssets: 1500, DebtRatio: 0.50, OperatingCash: 210, CapitalExpend: financialFloat(70)},
		{PeriodEnd: "2019-12-31", FiscalYear: 2019, Revenue: 1000, GrossMargin: 0.38, NetMargin: 0.15, CurrentAssets: 560, CurrentLiab: 260, TotalAssets: 1580, DebtRatio: 0.49, OperatingCash: 230, CapitalExpend: financialFloat(72)},
		{PeriodEnd: "2020-12-31", FiscalYear: 2020, Revenue: 1120, GrossMargin: 0.39, NetMargin: 0.155, CurrentAssets: 610, CurrentLiab: 275, TotalAssets: 1680, DebtRatio: 0.48, OperatingCash: 252, CapitalExpend: financialFloat(78)},
		{PeriodEnd: "2021-12-31", FiscalYear: 2021, Revenue: 1280, GrossMargin: 0.40, NetMargin: 0.16, CurrentAssets: 670, CurrentLiab: 290, TotalAssets: 1810, DebtRatio: 0.47, OperatingCash: 280, CapitalExpend: financialFloat(86)},
		{PeriodEnd: "2022-12-31", FiscalYear: 2022, Revenue: 1440, GrossMargin: 0.405, NetMargin: 0.165, CurrentAssets: 735, CurrentLiab: 305, TotalAssets: 1940, DebtRatio: 0.46, OperatingCash: 310, CapitalExpend: financialFloat(94), MissingInv: true},
		{PeriodEnd: "2023-12-31", FiscalYear: 2023, Revenue: 1600, GrossMargin: 0.41, NetMargin: 0.17, CurrentAssets: 810, CurrentLiab: 320, TotalAssets: 2080, DebtRatio: 0.45, OperatingCash: 345, CapitalExpend: financialFloat(105)},
	}
}

func quarterlyFinancialSeedValues() []financialSeedValue {
	values := make([]financialSeedValue, 0, 21)
	for year := 2019; year <= 2024; year++ {
		lastQuarter := 4
		if year == 2024 {
			lastQuarter = 1
		}
		for quarter := 1; quarter <= lastQuarter; quarter++ {
			periodEnd := quarterlyPeriodEnd(year, quarter)
			revenue := 215 + float64(year-2019)*25 + float64(quarter)*18
			capitalExpenditure := 18 + float64(year-2019)*1.5 + float64(quarter)
			var capitalExpendPointer *float64 = financialFloat(capitalExpenditure)
			if year == 2021 && quarter == 3 {
				capitalExpendPointer = nil
			}
			values = append(values, financialSeedValue{
				PeriodEnd: periodEnd, FiscalYear: year, FiscalQuarter: fmt.Sprintf("Q%d", quarter), Revenue: revenue,
				GrossMargin: 0.36 + float64(quarter)*0.008, NetMargin: 0.145 + float64(quarter)*0.004,
				CurrentAssets: 390 + float64(year-2019)*20 + float64(quarter)*8, CurrentLiab: 190 + float64(year-2019)*9 + float64(quarter)*4,
				TotalAssets: 1130 + float64(year-2019)*45 + float64(quarter)*15, DebtRatio: 0.52 - float64(year-2019)*0.012,
				OperatingCash: revenue * (0.19 + float64(quarter)*0.006), CapitalExpend: capitalExpendPointer,
			})
		}
	}
	return values
}

func buildFinancialReport(value financialSeedValue) stockdomain.FinancialReport {
	period := stockdomain.PeriodAnnual
	if value.FiscalQuarter != "" {
		period = stockdomain.PeriodQuarterly
	}
	totalLiabilities := value.TotalAssets * value.DebtRatio
	totalEquity := value.TotalAssets - totalLiabilities
	netProfit := value.Revenue * value.NetMargin
	capitalExpenditure := financialString(value.CapitalExpend)
	return stockdomain.FinancialReport{
		InstrumentCode: "000001.SZ", Period: period, PeriodEnd: value.PeriodEnd, FiscalYear: value.FiscalYear, FiscalQuarter: value.FiscalQuarter,
		PublishedAt: financialPublishedAt(value),
		Income:      stockdomain.IncomeStatement{Revenue: financialNumber(value.Revenue), GrossProfit: financialNumber(value.Revenue * value.GrossMargin), OperatingProfit: financialNumber(netProfit * 1.55), NetProfit: financialNumber(netProfit)},
		Balance:     stockdomain.BalanceSheet{CashAndEquivalents: financialNumber(value.CurrentAssets * 0.35), AccountsReceivable: financialNumber(value.CurrentAssets * 0.25), Inventory: financialNumberWithMissing(value.CurrentAssets*0.30, value.MissingInv), CurrentAssets: financialNumber(value.CurrentAssets), CurrentLiabilities: financialNumber(value.CurrentLiab), TotalAssets: financialNumber(value.TotalAssets), TotalLiabilities: financialNumber(totalLiabilities), TotalEquity: financialNumber(totalEquity)},
		CashFlow:    stockdomain.CashFlowStatement{OperatingCashFlow: financialNumber(value.OperatingCash), CapitalExpenditure: capitalExpenditure, InvestingCashFlow: financialNumber(-value.OperatingCash * 0.22), FinancingCashFlow: financialNumber(-value.OperatingCash * 0.10), NetCashChange: financialNumber(value.OperatingCash * 0.68)},
	}
}

func financialPublishedAt(value financialSeedValue) *string {
	date := value.FiscalYear + 1
	if value.FiscalQuarter != "" {
		date = value.FiscalYear
		if value.FiscalQuarter == "Q4" {
			date++
		}
	}
	monthDay := "04-30"
	if value.FiscalQuarter == "Q2" {
		monthDay = "08-30"
	} else if value.FiscalQuarter == "Q3" {
		monthDay = "10-30"
	}
	result := fmt.Sprintf("%d-%s", date, monthDay)
	return &result
}

func quarterlyPeriodEnd(year, quarter int) string {
	monthDays := map[int]string{1: "03-31", 2: "06-30", 3: "09-30", 4: "12-31"}
	return fmt.Sprintf("%d-%s", year, monthDays[quarter])
}

func financialNumber(value float64) *string {
	formatted := strconv.FormatFloat(value, 'f', 2, 64)
	return &formatted
}

func financialNumberWithMissing(value float64, missing bool) *string {
	if missing {
		return nil
	}
	return financialNumber(value)
}

func financialString(value *float64) *string {
	if value == nil {
		return nil
	}
	return financialNumber(*value)
}

func financialFloat(value float64) *float64 {
	return &value
}

func validateFinancialReports(fixture Fixture) error {
	instruments := make(map[string]struct{}, len(fixture.Instruments))
	for _, instrument := range fixture.Instruments {
		instruments[instrument.Code] = struct{}{}
	}
	seen := make(map[string]struct{}, len(fixture.FinancialReports))
	for _, report := range fixture.FinancialReports {
		if err := report.Validate(); err != nil {
			return fmt.Errorf("validate financial report %q/%q: %w", report.InstrumentCode, report.PeriodEnd, err)
		}
		if _, exists := instruments[report.InstrumentCode]; !exists {
			return fmt.Errorf("financial report references unknown instrument %q", report.InstrumentCode)
		}
		key := report.InstrumentCode + "\x00" + string(report.Period) + "\x00" + report.PeriodEnd
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate financial report %q/%q", report.InstrumentCode, report.PeriodEnd)
		}
		seen[key] = struct{}{}
	}
	return nil
}
