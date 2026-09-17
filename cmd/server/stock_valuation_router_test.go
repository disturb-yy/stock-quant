package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/disturb-yy/stock-quant/internal/stock"
	"github.com/gin-gonic/gin"
)

type fakeStockValuationReader struct {
	valuation stock.StockValuation
}

func (reader fakeStockValuationReader) Valuation(context.Context, stock.ValuationRequest) (stock.StockValuation, error) {
	return reader.valuation, nil
}

func TestNewRouterRegistersStockValuation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouterWithMarketSignalsAndRankingsAndStocksAndBarsAndFinancialsAndValuation(applicationLogger, nil, nil, nil, nil, nil, nil, nil, fakeStockValuationReader{
		valuation: stock.StockValuation{Symbol: "000001.SZ", Name: "平安银行", RequestedRange: stock.ValuationRangeFiveYears, Metrics: stock.ValuationMetrics{}},
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/stocks/000001.SZ/valuation?range=5y", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"requested_range":"5y"`) {
		t.Fatalf("status/body = %d/%s, want stock valuation", response.Code, response.Body.String())
	}
}
