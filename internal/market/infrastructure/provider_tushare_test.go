package infrastructure

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/disturb-yy/stock-quant/pkg/config"
)

func TestTushareClientQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload["api_name"] != "stock_basic" || payload["token"] != "secret-token" {
			t.Fatalf("request payload = %#v", payload)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"code":0,"msg":"","data":{"fields":["ts_code","name","total_mv"],"items":[["000001.SZ","平安银行",123.45],["600519.SH","贵州茅台",null]]}}`))
	}))
	defer server.Close()

	client, err := NewTushareClient(config.Tushare{Token: "secret-token", Endpoint: server.URL, Timeout: time.Second})
	if err != nil {
		t.Fatalf("NewTushareClient() error = %v", err)
	}
	rows, err := client.Query(context.Background(), "stock_basic", map[string]string{"list_status": "L"}, "ts_code,name,total_mv")
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if len(rows) != 2 || rows[0]["ts_code"] != "000001.SZ" || rows[0]["total_mv"] != "123.45" || rows[1]["total_mv"] != "" {
		t.Fatalf("Query() rows = %#v", rows)
	}
}

func TestNewTushareClientRequiresToken(t *testing.T) {
	if _, err := NewTushareClient(config.Tushare{Endpoint: "https://api.tushare.pro"}); err == nil {
		t.Fatal("NewTushareClient() error = nil, want token validation error")
	}
}

func TestTushareClientQueryAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"code":2002,"msg":"权限不足"}`))
	}))
	defer server.Close()
	client, err := NewTushareClient(config.Tushare{Token: "secret-token", Endpoint: server.URL})
	if err != nil {
		t.Fatalf("NewTushareClient() error = %v", err)
	}
	_, err = client.Query(context.Background(), "daily", nil, "")
	if err == nil || !strings.Contains(err.Error(), "code 2002") {
		t.Fatalf("Query() error = %v, want Tushare API error", err)
	}
}

func TestTushareTransformations(t *testing.T) {
	bar, err := buildTushareBar(map[string]string{
		"ts_code": "000001.SZ", "trade_date": "20260918", "open": "10.00", "high": "10.50", "low": "9.90", "close": "10.20", "vol": "123.5", "amount": "456.78",
	})
	if err != nil {
		t.Fatalf("buildTushareBar() error = %v", err)
	}
	if bar.TradeDate != "2026-09-18" || bar.Volume != 12350 || bar.TurnoverAmount != "456780.000000" {
		t.Fatalf("bar = %#v", bar)
	}

	basic, err := buildTushareBasic(map[string]string{
		"ts_code": "000001.SZ", "trade_date": "20260918", "total_mv": "100.5", "pb": "1.2", "pe_ttm": "10.1", "ps_ttm": "2.3", "turnover_rate": "3.4",
	})
	if err != nil {
		t.Fatalf("buildTushareBasic() error = %v", err)
	}
	if basic.MarketCap != "1005000.000000" || basic.PETTM != "10.100000" || basic.TurnoverRate != "3.400000" {
		t.Fatalf("basic = %#v", basic)
	}
	qfq, err := divideTushareNumber("108.031", "120.000", 8)
	if err != nil || qfq != "0.90025833" {
		t.Fatalf("divideTushareNumber() = %q, error = %v", qfq, err)
	}
}

func TestSyncDateRangeDefaultsAndValidation(t *testing.T) {
	start, end, dates, err := syncDateRange(TushareSyncOptions{LookbackDays: 2}, time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("syncDateRange() error = %v", err)
	}
	if start != "20260916" || end != "20260918" || len(dates) != 3 || dates[0] != "20260918" || dates[2] != "20260916" {
		t.Fatalf("syncDateRange() = %q, %q, %#v", start, end, dates)
	}
	if _, _, _, err := syncDateRange(TushareSyncOptions{StartDate: "20260919", EndDate: "20260918"}, time.Now()); err == nil {
		t.Fatal("syncDateRange() error = nil, want invalid range error")
	}
}
