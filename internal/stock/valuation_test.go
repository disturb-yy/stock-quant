package stock

import (
	"context"
	"errors"
	"strings"
	"testing"

	stockdomain "github.com/disturb-yy/stock-quant/internal/stock/domain"
)

type fakeValuationReader struct {
	snapshot ValuationSnapshot
	err      error
	request  ValuationRequest
}

func (reader *fakeValuationReader) ReadStockValuation(_ context.Context, request ValuationRequest) (ValuationSnapshot, error) {
	reader.request = request
	return reader.snapshot, reader.err
}

func TestValuationServiceBuildsRangePercentileAndIndustryMedian(t *testing.T) {
	reader := &fakeValuationReader{snapshot: valuationTestSnapshot()}
	service, err := NewValuationService(reader, ValuationSource{Mode: "demo", Provider: "mysql-demo-fixture"})
	if err != nil {
		t.Fatalf("NewValuationService() error = %v", err)
	}
	result, err := service.Valuation(context.Background(), ValuationRequest{Symbol: "000001.SZ", Range: ValuationRangeThreeYears})
	if err != nil {
		t.Fatalf("Valuation() error = %v", err)
	}
	if reader.request.Range != ValuationRangeThreeYears || result.EffectiveRange.From == nil || *result.EffectiveRange.From != "2020-06-26" || result.AsOf == nil || *result.AsOf != "2024-06-28" {
		t.Fatalf("request/range/as-of = %#v/%#v/%v, want 3y anchored by latest observation", reader.request, result.EffectiveRange, result.AsOf)
	}
	if len(result.Metrics.PETTM.History) != 4 || result.Metrics.PETTM.Percentile.SampleSize != 4 || result.Metrics.PETTM.Percentile.Method != "inclusive_rank" {
		t.Fatalf("PE history/percentile = %#v/%#v, want 4 observations and inclusive rank", result.Metrics.PETTM.History, result.Metrics.PETTM.Percentile)
	}
	if len(result.IndustryComparisons) != 1 || result.IndustryComparisons[0].Metrics.PETTM.SampleSize != 2 || result.IndustryComparisons[0].Metrics.PETTM.Value == nil || *result.IndustryComparisons[0].Metrics.PETTM.Value != "6.00" {
		t.Fatalf("industry comparison = %#v, want same-day two-peer median", result.IndustryComparisons)
	}
}

func TestValuationServicePreservesNegativeCurrentAndEmptyKnownStock(t *testing.T) {
	negative := "-1"
	reader := &fakeValuationReader{snapshot: valuationTestSnapshot()}
	reader.snapshot.Symbol = "300750.SZ"
	reader.snapshot.Name = "宁德时代"
	reader.snapshot.Observations = []stockdomain.ValuationObservation{{
		InstrumentCode: "300750.SZ", AsOf: "2024-06-28",
		PETTM: stockdomain.ValuationMetricObservation{AsOf: "2024-06-28", Value: &negative, Basis: "ttm"},
		PB:    stockdomain.ValuationMetricObservation{AsOf: "2024-06-28", Value: stringPointer("3.92"), Basis: "latest_daily_basic"},
		PSTTM: stockdomain.ValuationMetricObservation{AsOf: "2024-06-28"},
	}}
	service, err := NewValuationService(reader, ValuationSource{Mode: "demo", Provider: "mysql-demo-fixture"})
	if err != nil {
		t.Fatalf("NewValuationService() error = %v", err)
	}
	result, err := service.Valuation(context.Background(), ValuationRequest{Symbol: "300750.SZ"})
	if err != nil {
		t.Fatalf("Valuation() error = %v", err)
	}
	if result.Metrics.PETTM.Current.Value == nil || *result.Metrics.PETTM.Current.Value != "-1.00" || result.Metrics.PETTM.Percentile.Value != nil || result.Metrics.PETTM.Position != nil {
		t.Fatalf("negative PE result = %#v, want current only and null derived values", result.Metrics.PETTM)
	}

	reader.snapshot = valuationTestSnapshot()
	reader.snapshot.Symbol = "000002.SZ"
	reader.snapshot.Name = "万科A"
	reader.snapshot.Observations = nil
	reader.snapshot.Industries = nil
	result, err = service.Valuation(context.Background(), ValuationRequest{Symbol: "000002.SZ"})
	if err != nil {
		t.Fatalf("empty Valuation() error = %v", err)
	}
	if result.AsOf != nil || result.EffectiveRange.From != nil || result.Metrics.PETTM.Current.Value != nil || result.Metrics.PETTM.History == nil {
		t.Fatalf("empty valuation = %#v, want nullable empty structure", result)
	}
}

func TestValuationServiceValidatesRequestAndIdentity(t *testing.T) {
	reader := &fakeValuationReader{snapshot: valuationTestSnapshot()}
	service, err := NewValuationService(reader, ValuationSource{Mode: "demo", Provider: "mysql-demo-fixture"})
	if err != nil {
		t.Fatalf("NewValuationService() error = %v", err)
	}
	_, err = service.Valuation(context.Background(), ValuationRequest{Symbol: "bad symbol"})
	var validationError *ValuationValidationError
	if !errors.As(err, &validationError) {
		t.Fatalf("invalid request error = %v, want validation error", err)
	}
	reader.snapshot.Symbol = "300750.SZ"
	_, err = service.Valuation(context.Background(), ValuationRequest{Symbol: "000001.SZ"})
	if err == nil || !strings.Contains(err.Error(), "identity mismatch") {
		t.Fatalf("identity error = %v, want identity mismatch", err)
	}
}

func valuationTestSnapshot() ValuationSnapshot {
	return ValuationSnapshot{
		Symbol: "000001.SZ", Name: "平安银行", SeedVersion: "stk-004-demo-v1", SourceAsOf: "2024-06-28",
		Observations: []stockdomain.ValuationObservation{
			valuationObservation("000001.SZ", "2020-06-26", "4", "0.50", "1.00"),
			valuationObservation("000001.SZ", "2021-06-28", "5", "0.55", "1.10"),
			valuationObservation("000001.SZ", "2022-06-28", "6", "0.60", "1.20"),
			valuationObservation("000001.SZ", "2023-06-28", "7", "0.65", "1.30"),
			valuationObservation("000001.SZ", "2024-06-28", "8", "0.70", "1.40"),
		},
		Industries: []ValuationIndustrySnapshot{{Code: "BANK", Name: "银行", Members: []ValuationMemberSnapshot{
			{InstrumentCode: "601166.SH", Name: "兴业银行", Observations: []stockdomain.ValuationObservation{valuationObservation("601166.SH", "2024-06-28", "4", "0.80", "1.00")}},
			{InstrumentCode: "601398.SH", Name: "工商银行", Observations: []stockdomain.ValuationObservation{valuationObservation("601398.SH", "2024-06-28", "8", "0.90", "1.10")}},
		}}},
	}
}

func valuationObservation(symbol, asOf, pe, pb, ps string) stockdomain.ValuationObservation {
	return stockdomain.ValuationObservation{InstrumentCode: symbol, AsOf: asOf,
		PETTM: stockdomain.ValuationMetricObservation{AsOf: asOf, Value: stringPointer(pe), Basis: "ttm"},
		PB:    stockdomain.ValuationMetricObservation{AsOf: asOf, Value: stringPointer(pb), Basis: "latest_daily_basic"},
		PSTTM: stockdomain.ValuationMetricObservation{AsOf: asOf, Value: stringPointer(ps), Basis: "ttm"},
	}
}

func stringPointer(value string) *string {
	return &value
}
