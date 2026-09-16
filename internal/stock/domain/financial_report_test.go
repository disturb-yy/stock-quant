package domain

import (
	"fmt"
	"testing"
)

func TestSelectReportsRejectsDuplicateIdentity(t *testing.T) {
	report := domainTestReport(2024, "2024-12-31", "")
	_, err := SelectReports([]FinancialReport{report, report}, PeriodAnnual, RangeFiveYears)
	if err == nil {
		t.Fatal("SelectReports() error = nil, want duplicate period error")
	}
}

func TestCalculateIndicatorsRequiresSamePeriodComparison(t *testing.T) {
	current := domainTestReport(2024, "2024-03-31", "Q1")
	previousAnnual := domainTestReport(2023, "2023-12-31", "")
	indicators := CalculateIndicators(current, &previousAnnual)
	if indicators.RevenueYoYPct != nil || indicators.NetProfitYoYPct != nil {
		t.Fatalf("yoy = %#v/%#v, want nil for cross-period comparison", indicators.RevenueYoYPct, indicators.NetProfitYoYPct)
	}
}

func TestCanonicalCapitalExpenditureRejectsNegativeOrInvalidValues(t *testing.T) {
	negative := "-10"
	invalid := "not-a-number"
	valid := "10.50"
	if got := CanonicalCapitalExpenditure(&negative); got != nil {
		t.Fatalf("negative capital expenditure = %v, want nil", *got)
	}
	if got := CanonicalCapitalExpenditure(&invalid); got != nil {
		t.Fatalf("invalid capital expenditure = %v, want nil", *got)
	}
	if got := CanonicalCapitalExpenditure(&valid); got == nil || *got != valid {
		t.Fatalf("valid capital expenditure = %v, want %q", got, valid)
	}
}

func TestSelectReportsLimitsQuarterlyRangeAndPreservesMissingCashFlow(t *testing.T) {
	reports := make([]FinancialReport, 0, 16)
	for year := 2021; year <= 2024; year++ {
		for quarter := 1; quarter <= 4; quarter++ {
			reports = append(reports, domainTestReport(year, quarterlyDate(year, quarter), "Q"+string(rune('0'+quarter))))
		}
	}
	selected, err := SelectReports(reports, PeriodQuarterly, RangeThreeYears)
	if err != nil {
		t.Fatalf("SelectReports() error = %v", err)
	}
	if len(selected) != 12 || selected[0].PeriodEnd != "2022-01-01" || selected[len(selected)-1].PeriodEnd != "2024-10-01" {
		t.Fatalf("selected range = %d/%q..%q, want latest 12 quarters", len(selected), selected[0].PeriodEnd, selected[len(selected)-1].PeriodEnd)
	}
	current := domainTestReport(2024, "2024-10-01", "Q4")
	current.CashFlow.CapitalExpenditure = nil
	indicators := CalculateIndicators(current, nil)
	if indicators.FreeCashFlow != nil || indicators.GrossMarginPct == nil {
		t.Fatalf("indicators = %#v, want only FCF missing", indicators)
	}
}

func domainTestReport(year int, periodEnd, quarter string) FinancialReport {
	period := PeriodAnnual
	if quarter != "" {
		period = PeriodQuarterly
	}
	return FinancialReport{
		InstrumentCode: "000001.SZ", Period: period, PeriodEnd: periodEnd, FiscalYear: year, FiscalQuarter: quarter,
		Income:   IncomeStatement{Revenue: domainString("100"), GrossProfit: domainString("40"), NetProfit: domainString("20")},
		Balance:  BalanceSheet{CurrentAssets: domainString("100"), CurrentLiabilities: domainString("50"), TotalAssets: domainString("200"), TotalLiabilities: domainString("80"), TotalEquity: domainString("120")},
		CashFlow: CashFlowStatement{OperatingCashFlow: domainString("30"), CapitalExpenditure: domainString("10")},
	}
}

func quarterlyDate(year, quarter int) string {
	return fmt.Sprintf("%d-%s", year, map[int]string{1: "01-01", 2: "04-01", 3: "07-01", 4: "10-01"}[quarter])
}

func domainString(value string) *string {
	return &value
}
