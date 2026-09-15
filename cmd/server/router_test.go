package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/disturb-yy/stock-quant/internal/demo"
	"github.com/disturb-yy/stock-quant/internal/market"
	marketdomain "github.com/disturb-yy/stock-quant/internal/market/domain"
	"github.com/disturb-yy/stock-quant/pkg/api"
	"github.com/disturb-yy/stock-quant/pkg/config"
	"github.com/gin-gonic/gin"
)

func TestNewRouterRegistersHealthRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouter(applicationLogger)

	for _, path := range []string{"/api/v1/health"} {
		t.Run(path, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, path, nil)

			router.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
			}
		})
	}

	for _, path := range []string{"/heartbeat", "/probe", "/health", "/api/v1/dev/demo-status"} {
		t.Run("removed "+path, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, path, nil)

			router.ServeHTTP(response, request)

			if response.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
			}
		})
	}
}

type fakeDemoStatusReader struct {
	status demo.DemoStatus
}

func (reader fakeDemoStatusReader) DemoStatus(context.Context) (demo.DemoStatus, error) {
	return reader.status, nil
}

type fakeMarketOverviewReader struct {
	overview market.MarketOverview
	err      error
}

type fakeMarketSectorReader struct {
	sectors market.MarketSectors
}

type fakeMarketSignalReader struct {
	signals market.MarketSignals
}

type fakeAPISignalReader struct {
	snapshot market.SignalSnapshot
}

func (reader fakeAPISignalReader) ReadSignalSnapshot(context.Context, int) (market.SignalSnapshot, error) {
	return reader.snapshot, nil
}

func (reader fakeMarketSectorReader) Sectors(context.Context) (market.MarketSectors, error) {
	return reader.sectors, nil
}

func (reader fakeMarketSignalReader) Scan(context.Context, market.SignalRequest) (market.MarketSignals, error) {
	return reader.signals, nil
}

func (reader fakeMarketOverviewReader) Overview(context.Context) (market.MarketOverview, error) {
	return reader.overview, reader.err
}

func TestNewRouterRegistersMarketOverview(t *testing.T) {
	gin.SetMode(gin.TestMode)
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouterWithOverview(applicationLogger, fakeMarketOverviewReader{
		overview: market.MarketOverview{
			AsOf: "2024-06-28", ObservedAt: "2024-06-28T07:00:00Z",
			Breadth: market.BreadthOverview{Advancing: 2, Declining: 1},
		},
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/markets/overview", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), `"as_of":"2024-06-28"`) {
		t.Fatalf("response = %q, want market overview", response.Body.String())
	}
}

func TestNewRouterRegistersMarketSectors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouterWithMarket(applicationLogger, nil, fakeMarketSectorReader{sectors: market.MarketSectors{
		AsOf:    "2024-06-28",
		Sectors: []market.SectorOverview{{Code: "BANK", Name: "银行", ChangePercent: "0.88", ComponentCount: 1}},
	}})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/markets/sectors", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), `"code":"BANK"`) {
		t.Fatalf("response = %q, want market sectors", response.Body.String())
	}
}

func TestNewRouterRegistersMarketSignals(t *testing.T) {
	gin.SetMode(gin.TestMode)
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouterWithMarketAndSignals(applicationLogger, nil, nil, fakeMarketSignalReader{
		signals: market.MarketSignals{AsOf: "2024-06-28", Signals: []market.SignalResult{{Code: "300750.SZ", Name: "宁德时代"}}},
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/markets/signals?type=new_high", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), `"code":"300750.SZ"`) {
		t.Fatalf("response = %q, want market signals", response.Body.String())
	}
}

func TestMarketSignalsAPIIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service, err := market.NewSignalService(fakeAPISignalReader{snapshot: market.SignalSnapshot{
		SeedVersion: "seed", AsOf: "2024-01-21", Series: []marketdomain.SignalSeries{apiSignalSeries("A", "甲", 200, 100)},
	}}, market.ProviderSelection{Mode: market.ModeDemo, Provider: market.DemoProviderName})
	if err != nil {
		t.Fatalf("NewSignalService() error = %v", err)
	}
	server := httptest.NewServer(newRouterWithMarketAndSignals(applicationLogger, nil, nil, service))
	t.Cleanup(server.Close)

	tests := []struct {
		name       string
		path       string
		statusCode int
		code       api.ErrorCode
		wantBody   string
	}{
		{name: "success", path: "/api/v1/markets/signals?type=volume_surge&params=%7B%22window%22%3A20%2C%22multiple%22%3A2%7D", statusCode: http.StatusOK, wantBody: `"code":"A"`},
		{name: "invalid params", path: "/api/v1/markets/signals?type=volume_surge&params=%7B%22window%22%3A30%7D", statusCode: http.StatusBadRequest, code: api.CodeValidation},
		{name: "no match", path: "/api/v1/markets/signals?type=new_high", statusCode: http.StatusOK, wantBody: `"signals":[]`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := server.Client().Get(server.URL + test.path)
			if err != nil {
				t.Fatalf("GET market signals: %v", err)
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

func apiSignalSeries(code, name string, currentVolume, previousVolume int64) marketdomain.SignalSeries {
	bars := make([]marketdomain.DailyBar, 0, 21)
	for index := 0; index < 20; index++ {
		bars = append(bars, marketdomain.DailyBar{InstrumentCode: code, TradeDate: fmt.Sprintf("2024-01-%02d", index+1), Open: "99", High: "100", Low: "98", Close: "100", Volume: previousVolume, TurnoverAmount: "1"})
	}
	bars = append(bars, marketdomain.DailyBar{InstrumentCode: code, TradeDate: "2024-01-21", Open: "100", High: "101", Low: "99", Close: "100", Volume: currentVolume, TurnoverAmount: "1"})
	return marketdomain.SignalSeries{InstrumentCode: code, InstrumentName: name, Bars: bars}
}

func TestNewRouterRegistersDevelopmentDemoStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouter(applicationLogger, fakeDemoStatusReader{
		status: demo.DemoStatus{Mode: "demo", Provider: market.DemoProviderName, SeedVersion: demo.SeedVersion},
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/dev/demo-status", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), `"mode":"demo"`) {
		t.Fatalf("response = %q, want demo mode", response.Body.String())
	}

	openAPIResponse := httptest.NewRecorder()
	openAPIRequest := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
	router.ServeHTTP(openAPIResponse, openAPIRequest)
	if !strings.Contains(openAPIResponse.Body.String(), "/api/v1/dev/demo-status") {
		t.Fatal("development OpenAPI must include demo status path")
	}
}

func TestHealthEndpointOverHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httptest.NewServer(newRouter(applicationLogger))
	t.Cleanup(server.Close)

	response, err := server.Client().Get(server.URL + "/api/v1/health")
	if err != nil {
		t.Fatalf("GET /api/v1/health: %v", err)
	}
	t.Cleanup(func() { response.Body.Close() })

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read health response: %v", err)
	}
	if strings.TrimSpace(string(body)) != `{"status":"ok"}` {
		t.Fatalf("health body = %q, want %q", body, `{"status":"ok"}`)
	}
}

func TestOpenAPIEndpointOverHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httptest.NewServer(newRouter(applicationLogger))
	t.Cleanup(server.Close)

	response, err := server.Client().Get(server.URL + "/api/v1/openapi.json")
	if err != nil {
		t.Fatalf("GET /api/v1/openapi.json: %v", err)
	}
	t.Cleanup(func() { response.Body.Close() })

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if contentType := response.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("content type = %q, want application/json", contentType)
	}

	var document struct {
		OpenAPI string                     `json:"openapi"`
		Paths   map[string]json.RawMessage `json:"paths"`
	}
	if err := json.NewDecoder(response.Body).Decode(&document); err != nil {
		t.Fatalf("decode OpenAPI response: %v", err)
	}
	if document.OpenAPI != "3.0.3" {
		t.Fatalf("openapi = %q, want %q", document.OpenAPI, "3.0.3")
	}
	for _, path := range []string{"/api/v1/health", "/api/v1/markets/sectors", "/api/v1/markets/signals", "/api/v1/markets/rankings", "/api/v1/openapi.json"} {
		if _, ok := document.Paths[path]; !ok {
			t.Fatalf("OpenAPI response missing path %q", path)
		}
	}
}

func TestAPIV1ErrorsUseUnifiedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouter(applicationLogger)

	tests := []struct {
		name       string
		method     string
		path       string
		statusCode int
		code       api.ErrorCode
	}{
		{
			name:       "unknown route",
			method:     http.MethodGet,
			path:       "/api/v1/unknown",
			statusCode: http.StatusNotFound,
			code:       api.CodeNotFound,
		},
		{
			name:       "method not allowed",
			method:     http.MethodPost,
			path:       "/api/v1/health",
			statusCode: http.StatusMethodNotAllowed,
			code:       api.CodeMethodNotAllowed,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(test.method, test.path, nil)
			request.Header.Set("X-Request-ID", "contract-test-request")

			router.ServeHTTP(response, request)

			if response.Code != test.statusCode {
				t.Fatalf("status = %d, want %d", response.Code, test.statusCode)
			}
			if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
				t.Fatalf("content type = %q, want application/json", contentType)
			}
			if requestID := response.Header().Get("X-Request-ID"); requestID != "contract-test-request" {
				t.Fatalf("request ID = %q, want %q", requestID, "contract-test-request")
			}

			var body api.ErrorResponse
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if body.Code != test.code {
				t.Fatalf("error code = %q, want %q", body.Code, test.code)
			}
			if body.Message == "" {
				t.Fatal("error message must not be empty")
			}
		})
	}
}

func TestPaginationValidationMiddlewareIsRouteScoped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouter(applicationLogger)
	router.GET("/api/v1/contract-list", validatePaginationQuery(), func(context *gin.Context) {
		context.Status(http.StatusNoContent)
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/contract-list?page=0&page_size=101", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	var body api.ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Code != api.CodeInvalidPagination {
		t.Fatalf("error code = %q, want %q", body.Code, api.CodeInvalidPagination)
	}
}

func TestAPIV1PanicUsesSafeUnifiedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouter(applicationLogger)
	router.GET("/api/v1/panic", func(_ *gin.Context) {
		panic("sensitive panic detail")
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/panic", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	var body api.ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode recovery response: %v", err)
	}
	if body.Code != api.CodeInternal {
		t.Fatalf("error code = %q, want %q", body.Code, api.CodeInternal)
	}
	if strings.Contains(response.Body.String(), "sensitive panic detail") {
		t.Fatal("recovery response must not expose panic details")
	}
}

func TestNewHTTPServer(t *testing.T) {
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := newHTTPServer(applicationLogger, config.DefaultHTTPAddress)

	if server.Addr != config.DefaultHTTPAddress {
		t.Fatalf("server address = %q, want %q", server.Addr, config.DefaultHTTPAddress)
	}
	if server.Handler == nil {
		t.Fatal("server handler is nil")
	}
}
