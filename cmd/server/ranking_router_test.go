package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/disturb-yy/stock-quant/internal/market"
	marketdomain "github.com/disturb-yy/stock-quant/internal/market/domain"
	"github.com/disturb-yy/stock-quant/pkg/api"
	"github.com/gin-gonic/gin"
)

type fakeMarketRankingReader struct {
	rankings market.MarketRankings
}

func newTestServer(t *testing.T, router http.Handler) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return server
}

func (reader fakeMarketRankingReader) Rank(context.Context, market.RankingRequest) (market.MarketRankings, error) {
	return reader.rankings, nil
}

func TestNewRouterRegistersMarketRankings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouterWithMarketSignalsAndRankings(applicationLogger, nil, nil, nil, fakeMarketRankingReader{
		rankings: market.MarketRankings{Metric: marketdomain.RankingMetricGain, AsOf: "2024-06-28", Data: []market.RankingStock{{Code: "300750.SZ", Name: "宁德时代", Value: "21.00"}}},
	})

	response, err := http.Get(newTestServer(t, router).URL + "/api/v1/markets/rankings?metric=gain")
	if err != nil {
		t.Fatalf("GET market rankings: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if !strings.Contains(string(body), `"code":"300750.SZ"`) {
		t.Fatalf("response = %s, want ranking result", body)
	}
}

func TestMarketRankingsAPIIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service, err := market.NewRankingService(fakeRankingAPIReader{snapshot: market.RankingSnapshot{
		SeedVersion: "mkt-004-demo-v1", AsOf: "2024-06-28", Observations: []marketdomain.RankingObservation{
			{InstrumentCode: "A", InstrumentName: "甲", CurrentClose: "110", PreviousClose: "100", TurnoverAmount: "100", TurnoverRate: "1.00"},
			{InstrumentCode: "B", InstrumentName: "乙", CurrentClose: "90", PreviousClose: "100", TurnoverAmount: "200", TurnoverRate: "2.00"},
		}}}, market.ProviderSelection{Mode: market.ModeDemo, Provider: market.DemoProviderName})
	if err != nil {
		t.Fatalf("NewRankingService() error = %v", err)
	}
	server := newTestServer(t, newRouterWithMarketSignalsAndRankings(applicationLogger, nil, nil, nil, service))

	tests := []struct {
		name       string
		path       string
		statusCode int
		code       api.ErrorCode
		wantBody   string
	}{
		{name: "success", path: "/api/v1/markets/rankings?metric=loss&page=1&page_size=1", statusCode: http.StatusOK, wantBody: `"code":"B"`},
		{name: "invalid metric", path: "/api/v1/markets/rankings?metric=unknown", statusCode: http.StatusBadRequest, code: api.CodeValidation},
		{name: "invalid pagination", path: "/api/v1/markets/rankings?metric=gain&page_size=101", statusCode: http.StatusBadRequest, code: api.CodeInvalidPagination},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := server.Client().Get(server.URL + test.path)
			if err != nil {
				t.Fatalf("GET market rankings: %v", err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatalf("read response: %v", err)
			}
			if response.StatusCode != test.statusCode {
				t.Fatalf("status = %d, want %d; body = %s", response.StatusCode, test.statusCode, body)
			}
			if test.code != "" {
				var errorBody api.ErrorResponse
				if err := json.Unmarshal(body, &errorBody); err != nil {
					t.Fatalf("decode error response: %v", err)
				}
				if errorBody.Code != test.code {
					t.Fatalf("error code = %q, want %q", errorBody.Code, test.code)
				}
			}
			if test.wantBody != "" && !strings.Contains(string(body), test.wantBody) {
				t.Fatalf("body = %s, want substring %q", body, test.wantBody)
			}
		})
	}
}

type fakeRankingAPIReader struct {
	snapshot market.RankingSnapshot
}

func (reader fakeRankingAPIReader) ReadRankingSnapshot(context.Context) (market.RankingSnapshot, error) {
	return reader.snapshot, nil
}
