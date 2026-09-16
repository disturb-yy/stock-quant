package market

import (
	"context"
	"errors"
	"testing"

	"github.com/disturb-yy/stock-quant/internal/market/domain"
)

type fakeBarsReader struct {
	snapshot marketBarsSnapshot
	err      error
	request  BarsRequest
}

func (reader *fakeBarsReader) ReadStockBars(_ context.Context, request BarsRequest) (BarsSnapshot, error) {
	reader.request = request
	return reader.snapshot, reader.err
}

func TestBarsServiceBuildsAdjustedBarsAndBenchmark(t *testing.T) {
	reader := &fakeBarsReader{snapshot: BarsSnapshot{
		Symbol: "000001.SZ", Name: "平安银行", SeedVersion: "fnd-003-demo-v7",
		Bars: []domain.BarWithAdjustment{
			{Bar: barsDailyBar("2024-06-03", "10.00", 100), QFQFactor: "0.95", HFQFactor: "1.10"},
			{Bar: barsDailyBar("2024-06-04", "11.00", 200), QFQFactor: "1.00", HFQFactor: "1.20"},
		},
		BenchmarkCode: "000300.SH", BenchmarkName: "沪深300",
		Benchmark: []domain.BenchmarkBar{{TradeDate: "2024-06-03", Close: "2000"}, {TradeDate: "2024-06-04", Close: "2022"}},
	}}
	service, err := NewBarsService(reader, ProviderSelection{Mode: ModeDemo, Provider: DemoProviderName})
	if err != nil {
		t.Fatalf("NewBarsService() error = %v", err)
	}
	result, err := service.Bars(context.Background(), BarsRequest{Symbol: "000001.SZ", Adjust: "qfq", Range: "20d", Benchmark: "000300.SH"})
	if err != nil {
		t.Fatalf("Bars() error = %v", err)
	}
	if reader.request.Timeframe != "1d" || reader.request.Adjust != "qfq" || reader.request.Range != "20d" {
		t.Fatalf("reader request = %#v, want normalized chart request", reader.request)
	}
	if len(result.Bars) != 2 || result.Bars[0].Close != "9.50" || result.Bars[1].Close != "11.00" {
		t.Fatalf("bars = %#v, want qfq prices", result.Bars)
	}
	if result.Bars[0].Volume != 100 || result.Bars[1].Volume != 200 || result.Bars[0].MA5 != nil {
		t.Fatalf("bars volume/ma = %#v, want unchanged volume and nil short window MA", result.Bars)
	}
	if result.EffectiveRange.From == nil || *result.EffectiveRange.From != "2024-06-03" || result.EffectiveRange.To == nil || *result.EffectiveRange.To != "2024-06-04" {
		t.Fatalf("effective range = %#v, want actual returned dates", result.EffectiveRange)
	}
	if result.Benchmark == nil || len(result.Benchmark.Points) != 2 || result.Benchmark.Points[1].RelativeReturnPct != "14.69" {
		t.Fatalf("benchmark = %#v, want normalized common-date returns", result.Benchmark)
	}
	if result.Source.SeedVersion != "fnd-003-demo-v7" || result.Source.Mode != ModeDemo {
		t.Fatalf("source = %#v, want demo provenance", result.Source)
	}
}

func TestBarsServiceDefaultsAndEmptyResult(t *testing.T) {
	reader := &fakeBarsReader{snapshot: BarsSnapshot{Symbol: "000001.SZ", Name: "平安银行", SeedVersion: "seed", Bars: []domain.BarWithAdjustment{}}}
	service, err := NewBarsService(reader, ProviderSelection{Mode: ModeFallback, Provider: FallbackProviderName})
	if err != nil {
		t.Fatalf("NewBarsService() error = %v", err)
	}
	result, err := service.Bars(context.Background(), BarsRequest{Symbol: "000001.SZ"})
	if err != nil {
		t.Fatalf("Bars() error = %v", err)
	}
	if reader.request.Timeframe != "1d" || reader.request.Adjust != "none" || reader.request.Range != "120d" {
		t.Fatalf("default request = %#v, want 1d/none/120d", reader.request)
	}
	if result.Bars == nil || len(result.Bars) != 0 || result.EffectiveRange.From != nil || result.EffectiveRange.To != nil || result.Benchmark != nil {
		t.Fatalf("empty result = %#v, want empty bars and null benchmark/range", result)
	}
}

func TestBarsServiceRejectsInvalidRequestsAndPreservesReaderErrors(t *testing.T) {
	readerError := errors.New("database unavailable")
	service, err := NewBarsService(&fakeBarsReader{err: readerError}, ProviderSelection{Mode: ModeDemo, Provider: DemoProviderName})
	if err != nil {
		t.Fatalf("NewBarsService() error = %v", err)
	}
	tests := []struct {
		name    string
		request BarsRequest
	}{
		{name: "invalid symbol", request: BarsRequest{Symbol: "bad"}},
		{name: "unsupported timeframe", request: BarsRequest{Symbol: "000001.SZ", Timeframe: "1m"}},
		{name: "unsupported adjustment", request: BarsRequest{Symbol: "000001.SZ", Adjust: "split"}},
		{name: "range and dates conflict", request: BarsRequest{Symbol: "000001.SZ", Range: "20d", From: "2024-06-01", To: "2024-06-28"}},
		{name: "only one date", request: BarsRequest{Symbol: "000001.SZ", From: "2024-06-01"}},
		{name: "reversed dates", request: BarsRequest{Symbol: "000001.SZ", From: "2024-06-28", To: "2024-06-01"}},
		{name: "unsupported benchmark", request: BarsRequest{Symbol: "000001.SZ", Benchmark: "000001.SH"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := service.Bars(context.Background(), test.request); err == nil {
				t.Fatal("Bars() error = nil, want validation error")
			} else {
				var validationError *BarsValidationError
				if !errors.As(err, &validationError) {
					t.Fatalf("error = %v, want BarsValidationError", err)
				}
			}
		})
	}
	if _, err := service.Bars(context.Background(), BarsRequest{Symbol: "000001.SZ"}); !errors.Is(err, readerError) {
		t.Fatalf("reader error = %v, want %v", err, readerError)
	}
}

func TestBarsServiceMapsUnknownInstrument(t *testing.T) {
	service, err := NewBarsService(&fakeBarsReader{err: ErrBarsInstrumentNotFound}, ProviderSelection{Mode: ModeDemo, Provider: DemoProviderName})
	if err != nil {
		t.Fatalf("NewBarsService() error = %v", err)
	}
	_, err = service.Bars(context.Background(), BarsRequest{Symbol: "999999.SZ"})
	if !errors.Is(err, ErrBarsInstrumentNotFound) {
		t.Fatalf("error = %v, want ErrBarsInstrumentNotFound", err)
	}
}

func TestBarsServiceRejectsMissingAdjustmentFactor(t *testing.T) {
	service, err := NewBarsService(&fakeBarsReader{snapshot: BarsSnapshot{
		Symbol: "000001.SZ", Name: "平安银行", SeedVersion: "seed",
		Bars: []domain.BarWithAdjustment{{Bar: barsDailyBar("2024-06-03", "10", 1)}},
	}}, ProviderSelection{Mode: ModeDemo, Provider: DemoProviderName})
	if err != nil {
		t.Fatalf("NewBarsService() error = %v", err)
	}
	if _, err := service.Bars(context.Background(), BarsRequest{Symbol: "000001.SZ", Adjust: "qfq"}); err == nil {
		t.Fatal("Bars() error = nil, want dependency error for missing factor")
	}
}

func barsDailyBar(date, close string, volume int64) domain.DailyBar {
	return domain.DailyBar{InstrumentCode: "000001.SZ", TradeDate: date, Open: close, High: close, Low: close, Close: close, Volume: volume, TurnoverAmount: "1"}
}

// marketBarsSnapshot 只用于保持 fake 的字段初始化可读。
type marketBarsSnapshot = BarsSnapshot
