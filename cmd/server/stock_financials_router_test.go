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

type fakeStockFinancialsReader struct {
	financials stock.StockFinancials
}

func (reader fakeStockFinancialsReader) Financials(context.Context, stock.FinancialsRequest) (stock.StockFinancials, error) {
	return reader.financials, nil
}

func TestNewRouterRegistersStockFinancials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouterWithMarketSignalsAndRankingsAndStocksAndBarsAndFinancials(applicationLogger, nil, nil, nil, nil, nil, nil, fakeStockFinancialsReader{
		financials: stock.StockFinancials{Symbol: "000001.SZ", Name: "平安银行", Reports: []stock.FinancialReportView{}},
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/stocks/000001.SZ/financials?period=quarterly&range=3y", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"symbol":"000001.SZ"`) {
		t.Fatalf("status/body = %d/%s, want stock financials", response.Code, response.Body.String())
	}
}
