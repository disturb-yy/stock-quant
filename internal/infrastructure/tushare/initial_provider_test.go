package tushare

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"stock-quant/internal/shared/apperror"
	"stock-quant/internal/shared/types"
)

func TestInitialMarketDataProviderStockBasicsQueriesAndMapsOneStatus(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		var request struct {
			APIName string            `json:"api_name"`
			Params  map[string]string `json:"params"`
			Fields  string            `json:"fields"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		if request.APIName != "stock_basic" || request.Params["list_status"] != "D" {
			t.Errorf("request = %+v", request)
		}
		if request.Fields != "ts_code,symbol,name,area,industry,cnspell,market,exchange,list_status,list_date,delist_date" {
			t.Errorf("fields = %q", request.Fields)
		}
		_, _ = io.WriteString(w, `{"code":0,"msg":"","data":{"fields":["ts_code","symbol","name","area","industry","cnspell","market","exchange","list_status","list_date","delist_date"],"items":[["600000.SH","600000","浦发银行","上海","银行","PFYH","主板","SSE","D","19991110","20200102"]]}}`)
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, "token-secret", time.Second)
	provider := NewInitialMarketDataProvider(client)

	stocks, err := provider.StockBasics(context.Background(), "D")
	if err != nil {
		t.Fatalf("StockBasics() error = %v", err)
	}
	if requests != 1 || len(stocks) != 1 {
		t.Fatalf("requests/rows = %d/%d, want 1/1", requests, len(stocks))
	}
	stock := stocks[0]
	if stock.TSCode != "600000.SH" || stock.ListStatus != "D" || stock.Exchange != "SSE" || stock.ListDate.String() != "1999-11-10" || stock.DelistDate == nil || stock.DelistDate.String() != "2020-01-02" || stock.UpdatedAt.IsZero() {
		t.Fatalf("mapped stock = %+v", stock)
	}
}

func TestInitialMarketDataProviderStockBasicsRejectsInvalidStatusAndTooManyRows(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		_, _ = io.WriteString(w, stockBasicResponse(6001))
	}))
	defer server.Close()
	provider := NewInitialMarketDataProvider(newTestClient(t, server.URL, "token-secret", time.Second))

	if _, err := provider.StockBasics(context.Background(), "G"); err == nil || apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("invalid status error = %v", err)
	}
	if requests != 0 {
		t.Fatalf("invalid status sent %d requests", requests)
	}
	_, err := provider.StockBasics(context.Background(), "L")
	if err == nil || apperror.CodeOf(err) != apperror.CodeDataIncomplete || strings.Contains(err.Error(), "token-secret") {
		t.Fatalf("over-limit response error = %v", err)
	}
	if requests != 1 {
		t.Fatalf("stock_basic requests = %d, want 1", requests)
	}
}

func TestInitialMarketDataProviderTradeCalendarsSlicesNaturalYears(t *testing.T) {
	wantRanges := [][2]string{{"20200301", "20201231"}, {"20210101", "20211231"}, {"20220101", "20220228"}}
	calendarDates := []string{"20200302", "20210104", "20220225"}
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			APIName string            `json:"api_name"`
			Params  map[string]string `json:"params"`
			Fields  string            `json:"fields"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		if requests >= len(wantRanges) {
			t.Errorf("unexpected additional query: %+v", request)
			return
		}
		if request.APIName != "trade_cal" || request.Params["exchange"] != "SZSE" || request.Params["start_date"] != wantRanges[requests][0] || request.Params["end_date"] != wantRanges[requests][1] {
			t.Errorf("query %d = %+v, want %v", requests, request, wantRanges[requests])
		}
		if request.Fields != tradeCalendarFields {
			t.Errorf("fields = %q", request.Fields)
		}
		body := fmt.Sprintf(`{"code":0,"msg":"","data":{"fields":["cal_date","exchange","is_open","pretrade_date"],"items":[[%q,"SZSE",1,null]]}}`, calendarDates[requests])
		requests++
		_, _ = io.WriteString(w, body)
	}))
	defer server.Close()
	provider := NewInitialMarketDataProvider(newTestClient(t, server.URL, "token-secret", time.Second))
	from, _ := types.ParseTradingDate("2020-03-01")
	through, _ := types.ParseTradingDate("2022-02-28")

	calendars, err := provider.TradeCalendars(context.Background(), "SZSE", from, through)
	if err != nil {
		t.Fatalf("TradeCalendars() error = %v", err)
	}
	if requests != 3 || len(calendars) != 3 {
		t.Fatalf("requests/rows = %d/%d, want 3/3", requests, len(calendars))
	}
	for index, calendar := range calendars {
		if calendar.Exchange != "SZSE" || !calendar.IsOpen || calendar.Date.String() != []string{"2020-03-02", "2021-01-04", "2022-02-25"}[index] {
			t.Errorf("calendar[%d] = %+v", index, calendar)
		}
	}
}

func TestInitialMarketDataProviderRejectsUnsupportedCalendarInputAndPropagatesErrors(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		_, _ = io.WriteString(w, `{"code":2002,"msg":"token-secret permission denied","data":null}`)
	}))
	defer server.Close()
	provider := NewInitialMarketDataProvider(newTestClient(t, server.URL, "token-secret", time.Second))
	from, _ := types.ParseTradingDate("2020-01-01")
	through, _ := types.ParseTradingDate("2020-12-31")

	if _, err := provider.TradeCalendars(context.Background(), "SSE", through, from); err == nil || apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("reversed range error = %v", err)
	}
	if _, err := provider.TradeCalendars(context.Background(), "BJSE", from, through); err == nil || apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("unsupported exchange error = %v", err)
	}
	if requests != 0 {
		t.Fatalf("invalid inputs sent %d requests", requests)
	}
	_, err := provider.TradeCalendars(context.Background(), "SSE", from, through)
	if err == nil || apperror.CodeOf(err) != apperror.CodePermissionDenied || strings.Contains(err.Error(), "token-secret") {
		t.Fatalf("provider error = %v", err)
	}
	if requests != 1 {
		t.Fatalf("valid query count = %d, want 1", requests)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = provider.TradeCalendars(ctx, "SSE", from, through)
	if err == nil || apperror.CodeOf(err) != apperror.CodeCancelled || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled query error = %v", err)
	}
}

func stockBasicResponse(count int) string {
	const row = `["600000.SH","600000","浦发银行","上海","银行","PFYH","主板","SSE","L","19991110",""]`
	return `{"code":0,"msg":"","data":{"fields":["ts_code","symbol","name","area","industry","cnspell","market","exchange","list_status","list_date","delist_date"],"items":[` + strings.TrimSuffix(strings.Repeat(row+",", count), ",") + `]}}`
}
