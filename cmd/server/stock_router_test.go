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

type fakeStockOverviewReader struct {
	overview stock.StockOverview
}

func (reader fakeStockOverviewReader) Overview(context.Context, string) (stock.StockOverview, error) {
	return reader.overview, nil
}

func TestNewRouterRegistersStockOverview(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouterWithMarketSignalsAndRankingsAndStocks(logger, nil, nil, nil, nil, fakeStockOverviewReader{
		overview: stock.StockOverview{Symbol: "000001.SZ", Name: "平安银行", Industry: "银行"},
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/stocks/000001.SZ", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"symbol":"000001.SZ"`) {
		t.Fatalf("status/body = %d/%s, want stock overview", response.Code, response.Body.String())
	}
}
