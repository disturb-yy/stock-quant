package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/disturb-yy/stock-quant/internal/data/application"
	"github.com/disturb-yy/stock-quant/internal/data/domain"
	"github.com/gin-gonic/gin"
)

type fakeStockQueryService struct {
	result    domain.StockQueryResult
	err       error
	lastInput application.StockQueryInput
}

func (service *fakeStockQueryService) QueryStockData(_ context.Context,
	input application.StockQueryInput) (domain.StockQueryResult, error) {
	service.lastInput = input
	return service.result, service.err
}

func TestStockQueryHandlerReturnsEmptyPartialAndFullResults(t *testing.T) {
	updatedAt := time.Date(2026, 9, 26, 10, 20, 30, 0, time.UTC)
	dataAsOf := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	source := domain.SourceProvenance{Provider: "mock", Mode: "mock"}
	tests := []struct {
		name          string
		result        domain.StockQueryResult
		wantBasicNull bool
		wantBars      int
		wantStatus    int
	}{
		{name: "empty", result: domain.StockQueryResult{Symbol: "600519.SH", DailyBars: []domain.DailyBar{}, Availability: domain.StockDataAvailability{BasicInfo: domain.AvailabilityEmpty, DailyBars: domain.AvailabilityEmpty}}, wantBasicNull: true, wantStatus: http.StatusOK},
		{name: "partial", result: domain.StockQueryResult{Symbol: "600519.SH", BasicInfo: &domain.Instrument{Market: "SH", Name: "贵州茅台", Status: "normal"}, DailyBars: []domain.DailyBar{}, Availability: domain.StockDataAvailability{BasicInfo: domain.AvailabilityAvailable, DailyBars: domain.AvailabilityEmpty}, Source: &source, UpdatedAt: &updatedAt}, wantBars: 0, wantStatus: http.StatusOK},
		{name: "full", result: domain.StockQueryResult{Symbol: "600519.SH", BasicInfo: &domain.Instrument{Market: "SH", Name: "贵州茅台", Status: "normal"}, DailyBars: []domain.DailyBar{{TradeDate: dataAsOf, Open: 1, High: 2, Low: 1, Close: 1.5, Volume: 10}}, Availability: domain.StockDataAvailability{BasicInfo: domain.AvailabilityAvailable, DailyBars: domain.AvailabilityAvailable}, Source: &source, UpdatedAt: &updatedAt, DataAsOf: &dataAsOf}, wantBars: 1, wantStatus: http.StatusOK},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &fakeStockQueryService{result: test.result}
			recorder := performStockQueryRequest(service, "/api/v1/stocks/600519.SH/data")
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, test.wantStatus, recorder.Body.String())
			}
			var response stockDataResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if (response.BasicInfo == nil) != test.wantBasicNull || len(response.DailyBars) != test.wantBars {
				t.Fatalf("response = %#v", response)
			}
			if !strings.Contains(recorder.Body.String(), `"daily_bars":`) {
				t.Fatalf("daily_bars missing: %s", recorder.Body.String())
			}
		})
	}
}

func TestStockQueryHandlerPassesDatesAndMapsErrors(t *testing.T) {
	start := "2026-09-01"
	end := "2026-09-30"
	tests := []struct {
		name       string
		err        error
		path       string
		wantStatus int
		wantCode   string
	}{
		{name: "date parameters", path: "/api/v1/stocks/600519.SH/data?start_date=2026-09-01&end_date=2026-09-30", wantStatus: http.StatusOK},
		{name: "invalid request", err: domain.ErrInvalidDateRange, path: "/api/v1/stocks/600519.SH/data", wantStatus: http.StatusBadRequest, wantCode: "INVALID_REQUEST"},
		{name: "invalid symbol", err: domain.ErrInvalidStockSymbol, path: "/api/v1/stocks/bad/data", wantStatus: http.StatusBadRequest, wantCode: "INVALID_REQUEST"},
		{name: "stock not found", err: domain.ErrStockNotFound, path: "/api/v1/stocks/999999.SH/data", wantStatus: http.StatusNotFound, wantCode: "STOCK_NOT_FOUND"},
		{name: "dependency unavailable", err: domain.ErrDataSourceUnavailable, path: "/api/v1/stocks/600519.SH/data", wantStatus: http.StatusServiceUnavailable, wantCode: "DATA_SOURCE_UNAVAILABLE"},
		{name: "safe internal error", err: errors.New("sql DSN secret-token"), path: "/api/v1/stocks/600519.SH/data", wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_SERVER_ERROR"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &fakeStockQueryService{err: test.err, result: domain.StockQueryResult{Symbol: "600519.SH", DailyBars: []domain.DailyBar{}}}
			recorder := performStockQueryRequest(service, test.path)
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, test.wantStatus, recorder.Body.String())
			}
			if test.wantCode != "" {
				assertErrorCode(t, recorder, test.wantCode)
			}
			if test.name == "date parameters" {
				if service.lastInput.StartDate == nil || *service.lastInput.StartDate != start || service.lastInput.EndDate == nil || *service.lastInput.EndDate != end {
					t.Fatalf("date input = %#v", service.lastInput)
				}
			}
		})
	}
}

func performStockQueryRequest(service StockQueryService, path string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterStockQueryRoutes(router.Group("/api/v1"), service)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	router.ServeHTTP(recorder, request)
	return recorder
}
