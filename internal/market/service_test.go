package market

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/disturb-yy/stock-quant/internal/market/domain"
)

type fakeOverviewReader struct {
	snapshot OverviewSnapshot
	err      error
}

func (reader fakeOverviewReader) ReadOverviewSnapshot(context.Context) (OverviewSnapshot, error) {
	return reader.snapshot, reader.err
}

func TestOverviewReturnsStableContractAndIndexOrder(t *testing.T) {
	service, err := NewOverviewService(fakeOverviewReader{snapshot: validOverviewSnapshot()}, ProviderSelection{
		Mode: ModeDemo, Provider: DemoProviderName,
	})
	if err != nil {
		t.Fatalf("NewOverviewService() error = %v", err)
	}

	overview, err := service.Overview(context.Background())
	if err != nil {
		t.Fatalf("Overview() error = %v", err)
	}
	if overview.AsOf != "2024-06-28" || overview.ObservedAt != "2024-06-28T07:00:00Z" {
		t.Fatalf("observation point = %q/%q, want seed observation point", overview.AsOf, overview.ObservedAt)
	}
	if overview.Source.Mode != ModeDemo || overview.Source.Provider != DemoProviderName || overview.Source.SeedVersion != "fnd-003-demo-v3" {
		t.Fatalf("source = %#v, want demo seed source", overview.Source)
	}
	wantCodes := []string{"000001.SH", "399001.SZ", "399006.SZ", "000300.SH"}
	gotCodes := make([]string, 0, len(overview.Indices))
	for _, index := range overview.Indices {
		gotCodes = append(gotCodes, index.Code)
	}
	if !reflect.DeepEqual(gotCodes, wantCodes) {
		t.Fatalf("index order = %#v, want %#v", gotCodes, wantCodes)
	}
	if overview.Breadth.Advancing != 2 || overview.Breadth.Declining != 1 || overview.Turnover.Amount != "12002494200.00" {
		t.Fatalf("market stats = %#v/%#v, want seeded stats", overview.Breadth, overview.Turnover)
	}
}

func TestOverviewWrapsReaderError(t *testing.T) {
	wantErr := errors.New("database unavailable")
	service, err := NewOverviewService(fakeOverviewReader{err: wantErr}, ProviderSelection{Provider: DemoProviderName})
	if err != nil {
		t.Fatalf("NewOverviewService() error = %v", err)
	}
	if _, err := service.Overview(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("Overview() error = %v, want wrapped %v", err, wantErr)
	}
}

func TestOverviewRejectsIncompleteIndices(t *testing.T) {
	snapshot := validOverviewSnapshot()
	snapshot.Indices = snapshot.Indices[:3]
	service, err := NewOverviewService(fakeOverviewReader{snapshot: snapshot}, ProviderSelection{Provider: DemoProviderName})
	if err != nil {
		t.Fatalf("NewOverviewService() error = %v", err)
	}
	if _, err := service.Overview(context.Background()); err == nil {
		t.Fatal("Overview() error = nil, want missing index error")
	}
}

func validOverviewSnapshot() OverviewSnapshot {
	return OverviewSnapshot{
		SeedVersion: "fnd-003-demo-v3",
		AsOf:        "2024-06-28",
		ObservedAt:  "2024-06-28T07:00:00Z",
		Indices: []domain.IndexSnapshot{
			{Code: "000300.SH", Name: "沪深300", TradeDate: "2024-06-28", ObservedAt: "2024-06-28T07:00:00Z", Close: "3401.76", Change: "-5.87", ChangePercent: "-0.17"},
			{Code: "399006.SZ", Name: "创业板指", TradeDate: "2024-06-28", ObservedAt: "2024-06-28T07:00:00Z", Close: "1683.34", Change: "-8.12", ChangePercent: "-0.48"},
			{Code: "000001.SH", Name: "上证指数", TradeDate: "2024-06-28", ObservedAt: "2024-06-28T07:00:00Z", Close: "2994.73", Change: "-3.89", ChangePercent: "-0.13"},
			{Code: "399001.SZ", Name: "深证成指", TradeDate: "2024-06-28", ObservedAt: "2024-06-28T07:00:00Z", Close: "8848.42", Change: "-25.67", ChangePercent: "-0.29"},
		},
		Breadth: domain.MarketBreadth{
			TradeDate: "2024-06-28", Advancing: 2, Declining: 1, TurnoverAmount: "12002494200.00",
		},
	}
}
