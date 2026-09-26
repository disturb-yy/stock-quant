package source

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/disturb-yy/stock-quant/internal/data/config"
	"github.com/disturb-yy/stock-quant/internal/data/domain"
)

func TestMockAdapterIsDeterministicAndFiltersDates(t *testing.T) {
	adapter := NewMockAdapter()
	start := parseMockDate("2026-09-25")
	end := parseMockDate("2026-09-25")
	dateRange, err := domain.NewDateRange(&start, &end)
	if err != nil {
		t.Fatal(err)
	}
	first, err := adapter.FetchDailyBars(context.Background(), dateRange)
	if err != nil {
		t.Fatal(err)
	}
	second, err := adapter.FetchDailyBars(context.Background(), dateRange)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 2 || len(second.Items) != 2 || first.Items[0] != second.Items[0] {
		t.Fatalf("mock data is not deterministic: %#v %#v", first, second)
	}
	if adapter.Provenance() != (domain.SourceProvenance{Provider: "mock", Mode: "mock"}) {
		t.Fatalf("unexpected provenance: %#v", adapter.Provenance())
	}
}

func TestTushareAdapterConvertsBasicAndDailyResponses(t *testing.T) {
	var requests []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		requests = append(requests, payload)
		writer.Header().Set("Content-Type", "application/json")
		if payload["api_name"] == "stock_basic" {
			_, _ = writer.Write([]byte(`{"code":0,"data":{"fields":["ts_code","symbol","name","exchange","list_status"],"items":[["000001.SZ","000001","平安银行","SZ","L"],["000001.US","000001","Other","","L"]]}}`))
			return
		}
		_, _ = writer.Write([]byte(`{"code":0,"data":{"fields":["ts_code","trade_date","open","high","low","close","vol"],"items":[["000001.SZ","2026-09-25",10,11,9,10.5,1000]]}}`))
	}))
	defer server.Close()
	adapter, err := NewTushareAdapter(server.Client(), server.URL, "secret-token")
	if err != nil {
		t.Fatal(err)
	}
	info, err := adapter.FetchBasicInfo(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	wantAsOf := time.Now().UTC()
	wantAsOf = time.Date(wantAsOf.Year(), wantAsOf.Month(), wantAsOf.Day(), 0, 0, 0, 0, time.UTC)
	if len(info.Items) != 1 || info.Items[0].Market != "SZ" || info.DataAsOf == nil || !info.DataAsOf.Equal(wantAsOf) {
		t.Fatalf("unexpected basic info: %#v", info)
	}
	start := parseMockDate("2026-09-25")
	end := parseMockDate("2026-09-25")
	dateRange, err := domain.NewDateRange(&start, &end)
	if err != nil {
		t.Fatal(err)
	}
	bars, err := adapter.FetchDailyBars(context.Background(), dateRange)
	if err != nil {
		t.Fatal(err)
	}
	if len(bars.Items) != 1 || bars.Items[0].Close != 10.5 || bars.DataAsOf == nil {
		t.Fatalf("unexpected bars: %#v", bars)
	}
	if len(requests) != 2 || requests[0]["token"] != "secret-token" {
		t.Fatalf("unexpected requests: %#v", requests)
	}
}

func TestTushareAdapterDoesNotExposeTokenOnFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	adapter, err := NewTushareAdapter(server.Client(), server.URL, "secret-token")
	if err != nil {
		t.Fatal(err)
	}
	_, err = adapter.FetchBasicInfo(context.Background())
	if !errors.Is(err, domain.ErrDataSourceUnavailable) || strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("unexpected or unsafe error: %v", err)
	}
}

func TestTushareAdapterRequiresEndpointAndToken(t *testing.T) {
	if _, err := NewTushareAdapter(nil, "not-a-url", "token"); err == nil {
		t.Fatal("expected endpoint validation error")
	}
	if _, err := NewTushareAdapter(nil, "https://example.test", ""); err == nil {
		t.Fatal("expected token validation error")
	}
	if _, err := NewTushareAdapter(nil, "https://example.test", "token"); err != nil {
		t.Fatalf("unexpected valid adapter error: %v", err)
	}
}

func TestNewFromConfigSelectsConfiguredProvider(t *testing.T) {
	mock, err := NewFromConfig(config.Config{Provider: config.ProviderMock}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if mock.Provenance().Provider != config.ProviderMock {
		t.Fatalf("unexpected mock provider: %#v", mock.Provenance())
	}
	if _, err := NewFromConfig(config.Config{Provider: "unknown"}, nil); err == nil {
		t.Fatal("expected unsupported provider error")
	}
}
