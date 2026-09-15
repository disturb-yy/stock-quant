package market

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/disturb-yy/stock-quant/internal/market/domain"
)

type fakeSignalReader struct {
	snapshot SignalSnapshot
	err      error
	window   int
}

func (reader *fakeSignalReader) ReadSignalSnapshot(_ context.Context, window int) (SignalSnapshot, error) {
	reader.window = window
	return reader.snapshot, reader.err
}

func TestSignalServiceNormalizesRequestAndBuildsResponse(t *testing.T) {
	reader := &fakeSignalReader{snapshot: SignalSnapshot{
		SeedVersion: "fnd-003-demo-v4",
		AsOf:        "2024-01-21",
		Series:      []domain.SignalSeries{testMarketSignalSeries("300750.SZ", "宁德时代", 101, 150, 200, 100)},
	}}
	service, err := NewSignalService(reader, ProviderSelection{Mode: ModeDemo, Provider: DemoProviderName})
	if err != nil {
		t.Fatalf("NewSignalService() error = %v", err)
	}

	result, err := service.Scan(context.Background(), SignalRequest{
		Type:   " volume_surge ",
		Params: `{"window":20,"multiple":2}`,
	})
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if reader.window != 20 {
		t.Fatalf("reader window = %d, want 20", reader.window)
	}
	if result.Type != domain.SignalVolumeSurge || result.Params.Window != 20 || result.Params.Multiple == nil || *result.Params.Multiple != 2 {
		t.Fatalf("result identity/params = %#v, want normalized request", result)
	}
	if result.AsOf != "2024-01-21" || result.Source.SeedVersion != "fnd-003-demo-v4" || len(result.Signals) != 1 || result.Signals[0].Code != "300750.SZ" {
		t.Fatalf("result = %#v, want signal response", result)
	}
}

func TestSignalServiceDefaultsAndAcceptsStrongParameters(t *testing.T) {
	reader := &fakeSignalReader{snapshot: SignalSnapshot{
		SeedVersion: "seed",
		AsOf:        "2024-01-21",
		Series: []domain.SignalSeries{
			testMarketSignalSeries("A", "甲", 101, 101, 100, 100),
			testMarketSignalSeries("B", "乙", 130, 130, 100, 100),
		},
	}}
	service, err := NewSignalService(reader, ProviderSelection{Provider: DemoProviderName})
	if err != nil {
		t.Fatalf("NewSignalService() error = %v", err)
	}

	result, err := service.Scan(context.Background(), SignalRequest{Type: "strong", Params: `{"top_percent":20}`})
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if result.Params.Window != defaultSignalWindow || result.Params.TopPercent == nil || *result.Params.TopPercent != 20 || result.Params.Multiple != nil {
		t.Fatalf("params = %#v, want strong defaults and top percent", result.Params)
	}
	if len(result.Signals) != 1 || result.Signals[0].Code != "B" {
		t.Fatalf("signals = %#v, want strongest instrument", result.Signals)
	}
}

func TestSignalServiceRejectsInvalidRequests(t *testing.T) {
	service, err := NewSignalService(&fakeSignalReader{}, ProviderSelection{Provider: DemoProviderName})
	if err != nil {
		t.Fatalf("NewSignalService() error = %v", err)
	}
	tests := []struct {
		name    string
		request SignalRequest
		field   string
	}{
		{name: "missing type", request: SignalRequest{Params: `{}`}, field: "type"},
		{name: "unsupported type", request: SignalRequest{Type: "custom", Params: `{}`}, field: "type"},
		{name: "invalid window", request: SignalRequest{Type: "breakout", Params: `{"window":30}`}, field: "params.window"},
		{name: "invalid JSON", request: SignalRequest{Type: "breakout", Params: `{bad`}, field: "params"},
		{name: "irrelevant parameter", request: SignalRequest{Type: "strong", Params: `{"multiple":2}`}, field: "params.multiple"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.Scan(context.Background(), test.request)
			var validationError *SignalValidationError
			if !errors.As(err, &validationError) {
				t.Fatalf("error = %v, want SignalValidationError", err)
			}
			if _, ok := validationError.Fields[test.field]; !ok {
				t.Fatalf("fields = %#v, want %q", validationError.Fields, test.field)
			}
		})
	}
}

func TestSignalServiceReturnsInsufficientHistory(t *testing.T) {
	series := testMarketSignalSeries("A", "甲", 101, 101, 100, 100)
	series.Bars = series.Bars[:20]
	service, err := NewSignalService(&fakeSignalReader{snapshot: SignalSnapshot{
		SeedVersion: "seed", AsOf: "2024-01-20", Series: []domain.SignalSeries{series},
	}}, ProviderSelection{Provider: DemoProviderName})
	if err != nil {
		t.Fatalf("NewSignalService() error = %v", err)
	}
	_, err = service.Scan(context.Background(), SignalRequest{Type: "new_high"})
	var historyError *SignalHistoryError
	if !errors.As(err, &historyError) || historyError.Required != 21 || historyError.Available != 20 {
		t.Fatalf("error = %v, want insufficient history details", err)
	}
}

func TestSignalServiceWrapsReaderError(t *testing.T) {
	wantErr := errors.New("database unavailable")
	service, err := NewSignalService(&fakeSignalReader{err: wantErr}, ProviderSelection{Provider: DemoProviderName})
	if err != nil {
		t.Fatalf("NewSignalService() error = %v", err)
	}
	if _, err := service.Scan(context.Background(), SignalRequest{Type: "new_high"}); !errors.Is(err, wantErr) {
		t.Fatalf("Scan() error = %v, want wrapped %v", err, wantErr)
	}
}

func testMarketSignalSeries(code, name string, currentClose, currentHigh float64, volume, previousVolume int64) domain.SignalSeries {
	bars := make([]domain.DailyBar, 0, 21)
	for index := 0; index < 20; index++ {
		bars = append(bars, domain.DailyBar{
			InstrumentCode: code, TradeDate: testMarketSignalDate(index), Open: "99", High: "100", Low: "98", Close: "100",
			Volume: previousVolume, TurnoverAmount: "1",
		})
	}
	bars = append(bars, domain.DailyBar{
		InstrumentCode: code, TradeDate: testMarketSignalDate(20), Open: "100", High: testMarketSignalNumber(currentHigh), Low: "99", Close: testMarketSignalNumber(currentClose),
		Volume: volume, TurnoverAmount: "1",
	})
	return domain.SignalSeries{InstrumentCode: code, InstrumentName: name, Bars: bars}
}

func testMarketSignalDate(index int) string {
	return "2024-01-" + testMarketSignalDay(index+1)
}

func testMarketSignalDay(day int) string {
	if day < 10 {
		return "0" + string(rune('0'+day))
	}
	return string(rune('0'+day/10)) + string(rune('0'+day%10))
}

func testMarketSignalNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
