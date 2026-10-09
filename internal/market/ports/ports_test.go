package ports

import (
	"context"
	"testing"

	"stock-quant/internal/shared/types"
)

type fakeMarketDataProvider struct {
	from    types.TradingDate
	through types.TradingDate
	dates   []types.TradingDate
}

func (provider *fakeMarketDataProvider) TradingDates(_ context.Context, from, through types.TradingDate) ([]types.TradingDate, error) {
	provider.from = from
	provider.through = through
	return provider.dates, nil
}

var _ MarketDataProvider = (*fakeMarketDataProvider)(nil)

func TestMarketDataProviderFakePreservesRequestedRange(t *testing.T) {
	from, err := types.ParseTradingDate("2026-10-01")
	if err != nil {
		t.Fatal(err)
	}
	through, err := types.ParseTradingDate("2026-10-08")
	if err != nil {
		t.Fatal(err)
	}
	provider := &fakeMarketDataProvider{from: from, through: through, dates: []types.TradingDate{through}}
	dates, err := provider.TradingDates(context.Background(), from, through)
	if err != nil {
		t.Fatalf("TradingDates() error = %v", err)
	}
	if provider.from != from || provider.through != through || len(dates) != 1 || dates[0] != through {
		t.Fatalf("fake provider request/response = (%v, %v, %v)", provider.from, provider.through, dates)
	}
}
