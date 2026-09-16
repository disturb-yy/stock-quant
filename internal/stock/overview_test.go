package stock

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeOverviewReader struct {
	snapshot OverviewSnapshot
	err      error
	symbol   string
}

func (reader *fakeOverviewReader) ReadStockOverview(_ context.Context, symbol string) (OverviewSnapshot, error) {
	reader.symbol = symbol
	return reader.snapshot, reader.err
}

func TestOverviewServiceBuildsStockOverview(t *testing.T) {
	value := "5.82"
	reader := &fakeOverviewReader{snapshot: OverviewSnapshot{
		Symbol: "000001.SZ", Name: "平安银行", Industry: "银行",
		Quote:     QuoteSnapshot{Last: "10.31", Change: "0.09", ChangePct: "0.88", AsOf: "2024-06-28"},
		PETTM:     MetricSnapshot{Value: &value, AsOf: "2024-06-28", Basis: "ttm"},
		Sparkline: []SparklinePoint{{TradeDate: "2024-06-28", Open: "10.20", High: "10.40", Low: "10.10", Close: "10.31"}},
	}}
	service, err := NewOverviewService(reader)
	if err != nil {
		t.Fatalf("NewOverviewService() error = %v", err)
	}

	result, err := service.Overview(context.Background(), "000001.SZ")
	if err != nil {
		t.Fatalf("Overview() error = %v", err)
	}
	if reader.symbol != "000001.SZ" || result.Symbol != "000001.SZ" || result.Industry != "银行" {
		t.Fatalf("identity = %#v, reader symbol = %q", result, reader.symbol)
	}
	if result.Quote.AsOf != "2024-06-28" || result.Metrics.PETTM.Value == nil || *result.Metrics.PETTM.Value != "5.82" || result.Metrics.PETTM.AsOf == nil || *result.Metrics.PETTM.AsOf != "2024-06-28" {
		t.Fatalf("overview quote/metric = %#v/%#v", result.Quote, result.Metrics.PETTM)
	}
	if result.Sparkline.Period != "20d" || len(result.Sparkline.Points) != 1 {
		t.Fatalf("sparkline = %#v, want 20d point", result.Sparkline)
	}
	point := result.Sparkline.Points[0]
	if point.Open != "10.20" || point.High != "10.40" || point.Low != "10.10" || point.Close != "10.31" {
		t.Fatalf("sparkline OHLC = %#v, want source daily bar values", point)
	}
}

func TestOverviewServicePreservesMissingMetricsAndEmptySparkline(t *testing.T) {
	service, err := NewOverviewService(&fakeOverviewReader{snapshot: OverviewSnapshot{
		Symbol: "000001.SZ", Name: "平安银行", Industry: "银行",
		Quote:     QuoteSnapshot{Last: "10.31", Change: "0.09", ChangePct: "0.88", AsOf: "2024-06-28"},
		MarketCap: MetricSnapshot{AsOf: "2024-06-28", Basis: "latest_daily_basic"},
		Sparkline: []SparklinePoint{},
	}})
	if err != nil {
		t.Fatalf("NewOverviewService() error = %v", err)
	}

	result, err := service.Overview(context.Background(), "000001.SZ")
	if err != nil {
		t.Fatalf("Overview() error = %v", err)
	}
	if result.Metrics.MarketCap.Value != nil || result.Metrics.MarketCap.AsOf == nil || *result.Metrics.MarketCap.AsOf != "2024-06-28" || result.Metrics.MarketCap.Basis == nil || *result.Metrics.MarketCap.Basis != "latest_daily_basic" {
		t.Fatalf("missing metric = %#v, want null value with available metadata", result.Metrics.MarketCap)
	}
	if result.Sparkline.Points == nil || len(result.Sparkline.Points) != 0 {
		t.Fatalf("empty sparkline points = %#v, want non-nil empty list", result.Sparkline.Points)
	}
}

func TestOverviewServiceRejectsInvalidSymbol(t *testing.T) {
	reader := &fakeOverviewReader{}
	service, err := NewOverviewService(reader)
	if err != nil {
		t.Fatalf("NewOverviewService() error = %v", err)
	}
	_, err = service.Overview(context.Background(), "not a symbol")
	var validationError *SymbolValidationError
	if !errors.As(err, &validationError) {
		t.Fatalf("error = %v, want SymbolValidationError", err)
	}
	if reader.symbol != "" {
		t.Fatalf("reader symbol = %q, want no database call", reader.symbol)
	}
}

func TestOverviewServiceWrapsReaderError(t *testing.T) {
	wantErr := errors.New("database unavailable")
	service, err := NewOverviewService(&fakeOverviewReader{err: wantErr})
	if err != nil {
		t.Fatalf("NewOverviewService() error = %v", err)
	}
	_, err = service.Overview(context.Background(), "000001.SZ")
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrapped reader error", err)
	}
}

func TestOverviewServiceRejectsIdentityMismatch(t *testing.T) {
	service, err := NewOverviewService(&fakeOverviewReader{snapshot: OverviewSnapshot{Symbol: "300750.SZ"}})
	if err != nil {
		t.Fatalf("NewOverviewService() error = %v", err)
	}
	_, err = service.Overview(context.Background(), "000001.SZ")
	if err == nil || !strings.Contains(err.Error(), "identity mismatch") {
		t.Fatalf("error = %v, want identity mismatch", err)
	}
}
