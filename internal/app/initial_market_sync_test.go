package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"stock-quant/internal/market/domain"
	"stock-quant/internal/shared/types"
)

func TestInitialMarketSyncMergesHistoryAndStoresCalendars(t *testing.T) {
	listDate := syncDate(t, "2001-01-01")
	delistDate := syncDate(t, "2020-08-31")
	futureListDate := syncDate(t, "2027-01-01")
	provider := &initialSyncFakeProvider{
		stocks: map[string][]domain.Stock{
			"L": {{TSCode: "600001.SH", Name: "测试证券", Exchange: "SSE", Market: "主板", ListDate: listDate, ListStatus: "L", UpdatedAt: time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)}},
			"D": {{TSCode: "600001.SH", Name: "测试证券", Exchange: "SSE", Market: "主板", ListDate: listDate, DelistDate: &delistDate, ListStatus: "D", UpdatedAt: time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)}},
			"P": {
				{TSCode: "600001.SH", Name: "测试证券", Exchange: "SSE", Market: "主板", ListDate: listDate, ListStatus: "P", UpdatedAt: time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)},
				{TSCode: "600002.SH", Name: "待上市证券", Exchange: "SSE", Market: "主板", ListDate: futureListDate, ListStatus: "P", UpdatedAt: time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)},
			},
		},
		calendars: map[string][]domain.TradeCalendar{
			"SSE": {
				{Exchange: "SSE", Date: syncDate(t, "2026-10-08"), IsOpen: true},
				{Exchange: "SSE", Date: syncDate(t, "2026-10-09"), IsOpen: false},
			},
			"SZSE": {
				{Exchange: "SZSE", Date: syncDate(t, "2026-10-08"), IsOpen: true},
				{Exchange: "SZSE", Date: syncDate(t, "2026-10-09"), IsOpen: false},
			},
		},
	}
	stocks := &initialSyncFakeStockRepository{}
	calendar := &initialSyncFakeCalendarRepository{}
	service := NewInitialMarketSync(provider, stocks, calendar)

	result, err := service.Sync(context.Background(), syncDate(t, "2026-10-08"), syncDate(t, "2026-10-09"))
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if result.StockCount != 2 || result.CalendarCount != 4 {
		t.Fatalf("Sync() result = %#v, want 2 stocks and 4 calendar rows", result)
	}
	if len(stocks.items) != 2 {
		t.Fatalf("stored unique stocks = %d, want 2", len(stocks.items))
	}
	merged := stocks.items["600001.SH"]
	if merged.ListStatus != "D" || merged.DelistDate == nil || *merged.DelistDate != delistDate {
		t.Fatalf("merged historical stock = %#v, want D status and delist date", merged)
	}
	if !merged.ListedOn(syncDate(t, "2020-08-31")) || merged.ListedOn(syncDate(t, "2020-09-01")) {
		t.Fatal("delisted stock visibility must include its delist date and stop afterwards")
	}
	if stocks.items["600002.SH"].ListedOn(syncDate(t, "2026-10-09")) {
		t.Fatal("pre-listing stock must not be visible before list_date")
	}
	if got := provider.calls; len(got) != 5 || got[0] != "stock:L" || got[1] != "stock:D" || got[2] != "stock:P" || got[3] != "calendar:SSE" || got[4] != "calendar:SZSE" {
		t.Fatalf("provider calls = %v, want ordered stock statuses and both calendars only", got)
	}
	if len(calendar.items) != 4 || calendar.items[1].IsOpen {
		t.Fatalf("calendar rows = %#v, want all dates including the holiday", calendar.items)
	}
}

func TestInitialMarketSyncRejectsDelistedStockWithoutDelistDate(t *testing.T) {
	listDate := syncDate(t, "2001-01-01")
	provider := &initialSyncFakeProvider{stocks: map[string][]domain.Stock{
		"L": {},
		"D": {{TSCode: "600001.SH", Name: "已退市证券", Exchange: "SSE", Market: "主板", ListDate: listDate, ListStatus: "D", UpdatedAt: time.Now().UTC()}},
		"P": {},
	}}
	stocks := &initialSyncFakeStockRepository{}
	calendars := &initialSyncFakeCalendarRepository{}
	_, err := NewInitialMarketSync(provider, stocks, calendars).Sync(context.Background(), syncDate(t, "2020-01-01"), syncDate(t, "2020-01-02"))
	if err == nil {
		t.Fatal("Sync() error = nil for delisted stock without delist date")
	}
	if len(stocks.items) != 0 || len(calendars.items) != 0 {
		t.Fatalf("writes occurred for incomplete delisted stock: stocks=%d calendars=%d", len(stocks.items), len(calendars.items))
	}
}

func TestInitialMarketSyncRejectsCalendarMissingDateBeforeWrites(t *testing.T) {
	provider := &initialSyncFakeProvider{
		stocks: map[string][]domain.Stock{"L": {}, "D": {}, "P": {}},
		calendars: map[string][]domain.TradeCalendar{
			"SSE":  {{Exchange: "SSE", Date: syncDate(t, "2026-10-08"), IsOpen: true}},
			"SZSE": {{Exchange: "SZSE", Date: syncDate(t, "2026-10-08"), IsOpen: true}, {Exchange: "SZSE", Date: syncDate(t, "2026-10-09"), IsOpen: false}},
		},
	}
	stocks := &initialSyncFakeStockRepository{}
	calendars := &initialSyncFakeCalendarRepository{}
	_, err := NewInitialMarketSync(provider, stocks, calendars).Sync(context.Background(), syncDate(t, "2026-10-08"), syncDate(t, "2026-10-09"))
	if err == nil {
		t.Fatal("Sync() error = nil for incomplete SSE calendar")
	}
	if len(stocks.items) != 0 || len(calendars.items) != 0 {
		t.Fatalf("writes occurred for incomplete calendar: stocks=%d calendars=%d", len(stocks.items), len(calendars.items))
	}
}

func TestInitialMarketSyncRejectsConflictingListingDatesBeforeWrites(t *testing.T) {
	firstDate := syncDate(t, "2001-01-01")
	secondDate := syncDate(t, "2001-01-02")
	provider := &initialSyncFakeProvider{stocks: map[string][]domain.Stock{
		"L": {{TSCode: "600001.SH", Name: "证券", Exchange: "SSE", Market: "主板", ListDate: firstDate, ListStatus: "L", UpdatedAt: time.Now().UTC()}},
		"D": {{TSCode: "600001.SH", Name: "证券", Exchange: "SSE", Market: "主板", ListDate: secondDate, ListStatus: "D", UpdatedAt: time.Now().UTC()}},
		"P": {},
	}}
	stocks := &initialSyncFakeStockRepository{}
	calendar := &initialSyncFakeCalendarRepository{}
	service := NewInitialMarketSync(provider, stocks, calendar)

	_, err := service.Sync(context.Background(), syncDate(t, "2020-01-01"), syncDate(t, "2020-01-02"))
	if err == nil {
		t.Fatal("Sync() error = nil for conflicting list dates")
	}
	if len(stocks.items) != 0 || len(calendar.items) != 0 {
		t.Fatalf("writes occurred after source conflict: stocks=%d calendars=%d", len(stocks.items), len(calendar.items))
	}
}

func TestInitialMarketSyncProviderFailureDoesNotWriteRepositories(t *testing.T) {
	wantErr := errors.New("provider unavailable")
	provider := &initialSyncFakeProvider{stockErr: map[string]error{"P": wantErr}}
	stocks := &initialSyncFakeStockRepository{}
	calendar := &initialSyncFakeCalendarRepository{}
	service := NewInitialMarketSync(provider, stocks, calendar)

	_, err := service.Sync(context.Background(), syncDate(t, "2020-01-01"), syncDate(t, "2020-01-02"))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Sync() error = %v, want wrapped provider error", err)
	}
	if len(stocks.items) != 0 || len(calendar.items) != 0 {
		t.Fatalf("writes occurred after provider failure: stocks=%d calendars=%d", len(stocks.items), len(calendar.items))
	}
}

func TestInitialMarketSyncRejectsInvalidRangeWithoutProviderCalls(t *testing.T) {
	provider := &initialSyncFakeProvider{}
	service := NewInitialMarketSync(provider, &initialSyncFakeStockRepository{}, &initialSyncFakeCalendarRepository{})
	_, err := service.Sync(context.Background(), syncDate(t, "2020-01-02"), syncDate(t, "2020-01-01"))
	if err == nil || len(provider.calls) != 0 {
		t.Fatalf("Sync() error=%v calls=%v; want range error without I/O", err, provider.calls)
	}
}

type initialSyncFakeProvider struct {
	stocks    map[string][]domain.Stock
	calendars map[string][]domain.TradeCalendar
	stockErr  map[string]error
	calls     []string
}

func (f *initialSyncFakeProvider) StockBasics(_ context.Context, status string) ([]domain.Stock, error) {
	f.calls = append(f.calls, "stock:"+status)
	if err := f.stockErr[status]; err != nil {
		return nil, err
	}
	return append([]domain.Stock(nil), f.stocks[status]...), nil
}

func (f *initialSyncFakeProvider) TradeCalendars(_ context.Context, exchange string, _, _ types.TradingDate) ([]domain.TradeCalendar, error) {
	f.calls = append(f.calls, "calendar:"+exchange)
	return append([]domain.TradeCalendar(nil), f.calendars[exchange]...), nil
}

type initialSyncFakeStockRepository struct {
	items map[string]domain.Stock
	err   error
}

func (f *initialSyncFakeStockRepository) Upsert(_ context.Context, stocks []domain.Stock) error {
	if f.err != nil {
		return f.err
	}
	if f.items == nil {
		f.items = make(map[string]domain.Stock)
	}
	for _, stock := range stocks {
		f.items[stock.TSCode] = stock
	}
	return nil
}

func (f *initialSyncFakeStockRepository) Find(_ context.Context, code string) (domain.Stock, error) {
	stock, ok := f.items[code]
	if !ok {
		return domain.Stock{}, errors.New("stock not found")
	}
	return stock, nil
}

type initialSyncFakeCalendarRepository struct {
	items []domain.TradeCalendar
	err   error
}

func (f *initialSyncFakeCalendarRepository) Upsert(_ context.Context, rows []domain.TradeCalendar) error {
	if f.err != nil {
		return f.err
	}
	f.items = append([]domain.TradeCalendar(nil), rows...)
	return nil
}

func (f *initialSyncFakeCalendarRepository) List(_ context.Context, _ string, _, _ types.TradingDate) ([]domain.TradeCalendar, error) {
	return append([]domain.TradeCalendar(nil), f.items...), nil
}

func syncDate(t *testing.T, value string) types.TradingDate {
	t.Helper()
	date, err := types.ParseTradingDate(value)
	if err != nil {
		t.Fatalf("parse date: %v", err)
	}
	return date
}
