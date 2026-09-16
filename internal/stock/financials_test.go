package stock

import (
	"context"
	"testing"

	stockdomain "github.com/disturb-yy/stock-quant/internal/stock/domain"
)

type fakeFinancialsReader struct {
	snapshot FinancialsSnapshot
	err      error
	request  FinancialsRequest
}

func (reader *fakeFinancialsReader) ReadStockFinancials(_ context.Context, request FinancialsRequest) (FinancialsSnapshot, error) {
	reader.request = request
	return reader.snapshot, reader.err
}

func TestFinancialsServiceDefaultsAndBuildsLatestSummary(t *testing.T) {
	reader := &fakeFinancialsReader{snapshot: FinancialsSnapshot{
		Symbol: "000001.SZ", Name: "平安银行", SeedVersion: "fnd-003-demo-v7", AsOf: "2024-06-28",
		Reports: []stockdomain.FinancialReport{
			financialReport("2023-12-31", 2023, "1000", "400", "200", "100", "500", "250", "1000", "400", "600", "160", "50"),
			financialReport("2024-12-31", 2024, "1250", "500", "250", "120", "600", "300", "1200", "480", "720", "200", "80"),
		},
	}}
	service, err := NewFinancialsService(reader, FinancialSource{Mode: "demo", Provider: "mysql-demo-fixture"})
	if err != nil {
		t.Fatalf("NewFinancialsService() error = %v", err)
	}

	result, err := service.Financials(context.Background(), FinancialsRequest{Symbol: "000001.SZ"})
	if err != nil {
		t.Fatalf("Financials() error = %v", err)
	}
	if reader.request.Period != FinancialPeriodAnnual || reader.request.Range != FinancialRangeFiveYears {
		t.Fatalf("reader request = %#v, want annual/5y defaults", reader.request)
	}
	if len(result.Reports) != 2 || result.Reports[0].PeriodEnd != "2023-12-31" || result.Reports[1].PeriodEnd != "2024-12-31" {
		t.Fatalf("reports = %#v, want ascending report periods", result.Reports)
	}
	if result.Summary == nil || result.Summary.PeriodEnd != "2024-12-31" {
		t.Fatalf("summary = %#v, want latest report", result.Summary)
	}
	if result.Summary.RevenueYoYPct == nil || *result.Summary.RevenueYoYPct != "25.00" || result.Summary.NetProfitYoYPct == nil || *result.Summary.NetProfitYoYPct != "20.00" {
		t.Fatalf("summary yoy = %#v, want 25.00/20.00", result.Summary)
	}
	if result.Summary.GrossMarginPct == nil || *result.Summary.GrossMarginPct != "40.00" || result.Summary.ROEPct == nil || *result.Summary.ROEPct != "16.67" {
		t.Fatalf("summary profitability = %#v, want 40.00/16.67", result.Summary)
	}
	if result.Summary.FreeCashFlow == nil || *result.Summary.FreeCashFlow != "120.00" || result.Summary.CurrentRatio == nil || *result.Summary.CurrentRatio != "2.00" {
		t.Fatalf("summary cash/quality = %#v, want 120.00/2.00", result.Summary)
	}
	if result.Source.AsOf != "2024-06-28" || result.Source.SeedVersion != "fnd-003-demo-v7" {
		t.Fatalf("source = %#v, want snapshot provenance", result.Source)
	}
}

func financialReport(periodEnd string, fiscalYear int, revenue, grossProfit, operatingProfit, netProfit, currentAssets, currentLiabilities, totalAssets, totalLiabilities, totalEquity, operatingCashFlow, capitalExpenditure string) stockdomain.FinancialReport {
	return stockdomain.FinancialReport{
		InstrumentCode: "000001.SZ", Period: stockdomain.PeriodAnnual, PeriodEnd: periodEnd, FiscalYear: fiscalYear,
		Income:   stockdomain.IncomeStatement{Revenue: financialStringPtr(revenue), GrossProfit: financialStringPtr(grossProfit), OperatingProfit: financialStringPtr(operatingProfit), NetProfit: financialStringPtr(netProfit)},
		Balance:  stockdomain.BalanceSheet{CurrentAssets: financialStringPtr(currentAssets), CurrentLiabilities: financialStringPtr(currentLiabilities), TotalAssets: financialStringPtr(totalAssets), TotalLiabilities: financialStringPtr(totalLiabilities), TotalEquity: financialStringPtr(totalEquity)},
		CashFlow: stockdomain.CashFlowStatement{OperatingCashFlow: financialStringPtr(operatingCashFlow), CapitalExpenditure: financialStringPtr(capitalExpenditure)},
	}
}

func financialStringPtr(value string) *string {
	return &value
}
