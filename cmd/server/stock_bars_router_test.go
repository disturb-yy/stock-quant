package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/disturb-yy/stock-quant/internal/market"
	"github.com/gin-gonic/gin"
)

type fakeStockBarsReader struct {
	bars market.StockBars
}

func (reader fakeStockBarsReader) Bars(context.Context, market.BarsRequest) (market.StockBars, error) {
	return reader.bars, nil
}

func TestNewRouterRegistersStockBars(t *testing.T) {
	gin.SetMode(gin.TestMode)
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouterWithMarketSignalsAndRankingsAndStocksAndBars(applicationLogger, nil, nil, nil, nil, nil, fakeStockBarsReader{
		bars: market.StockBars{Symbol: "000001.SZ", Name: "平安银行", Timeframe: "1d", Bars: []market.StockBar{}},
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/stocks/000001.SZ/bars?range=20d", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"timeframe":"1d"`) {
		t.Fatalf("status/body = %d/%s, want stock bars", response.Code, response.Body.String())
	}
}
