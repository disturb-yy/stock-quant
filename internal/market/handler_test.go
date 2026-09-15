package market

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/disturb-yy/stock-quant/internal/market/domain"
	"github.com/disturb-yy/stock-quant/pkg/api"
	"github.com/gin-gonic/gin"
)

type fakeOverviewQuery struct {
	overview MarketOverview
	err      error
}

type fakeSectorQuery struct {
	sectors MarketSectors
	err     error
}

type fakeSignalScanner struct {
	result  MarketSignals
	err     error
	request SignalRequest
}

func (query fakeSectorQuery) Sectors(context.Context) (MarketSectors, error) {
	return query.sectors, query.err
}

func (query fakeOverviewQuery) Overview(context.Context) (MarketOverview, error) {
	return query.overview, query.err
}

func (scanner *fakeSignalScanner) Scan(_ context.Context, request SignalRequest) (MarketSignals, error) {
	scanner.request = request
	return scanner.result, scanner.err
}

func TestRegisterRoutesReturnsMarketOverview(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), fakeOverviewQuery{overview: MarketOverview{
		AsOf: "2024-06-28", ObservedAt: "2024-06-28T07:00:00Z",
		Indices: []IndexOverview{{Code: "000001.SH", Name: "上证指数", Close: "2994.73", Change: "-3.89", ChangePercent: "-0.13"}},
		Breadth: BreadthOverview{Advancing: 2, Declining: 1}, Turnover: TurnoverOverview{Amount: "12002494200.00", Currency: "CNY"},
	}})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/markets/overview", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var body MarketOverview
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.AsOf != "2024-06-28" || body.Turnover.Currency != "CNY" {
		t.Fatalf("body = %#v, want market overview", body)
	}
}

func TestRegisterRoutesReturnsSafeDependencyError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), fakeOverviewQuery{err: errors.New("secret database detail")})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/markets/overview", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if strings.Contains(response.Body.String(), "secret database detail") {
		t.Fatal("dependency error must not be exposed")
	}
	var body api.ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Code != api.CodeDependencyUnavailable {
		t.Fatalf("error code = %q, want %q", body.Code, api.CodeDependencyUnavailable)
	}
}

func TestRegisterSectorRoutesReturnsMarketSectors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterSectorRoutes(router.Group("/api/v1"), fakeSectorQuery{sectors: MarketSectors{
		AsOf: "2024-06-28", Source: DataSource{Mode: ModeDemo, Provider: DemoProviderName, SeedVersion: "mkt-002-demo-v1"},
		Sectors: []SectorOverview{{Code: "BANK", Name: "银行", ChangePercent: "0.88", ComponentCount: 1, Leader: SectorLeaderOverview{Code: "000001.SZ", Name: "平安银行", ChangePercent: "0.88"}}},
	}})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/markets/sectors", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var body MarketSectors
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.AsOf != "2024-06-28" || len(body.Sectors) != 1 || body.Sectors[0].Leader.Code != "000001.SZ" {
		t.Fatalf("body = %#v, want market sectors", body)
	}
}

func TestRegisterSectorRoutesReturnsSafeDependencyError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterSectorRoutes(router.Group("/api/v1"), fakeSectorQuery{err: errors.New("secret database detail")})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/markets/sectors", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if strings.Contains(response.Body.String(), "secret database detail") {
		t.Fatal("dependency error must not be exposed")
	}
	var body api.ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Code != api.CodeDependencyUnavailable {
		t.Fatalf("error code = %q, want %q", body.Code, api.CodeDependencyUnavailable)
	}
}

func TestRegisterSignalRoutesReturnsSignalsAndDecodesQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	scanner := &fakeSignalScanner{result: MarketSignals{
		Type:   domain.SignalVolumeSurge,
		Params: SignalParameters{Window: 20},
		AsOf:   "2024-01-21", Signals: []SignalResult{{Code: "A", Name: "甲", Signal: domain.SignalVolumeSurge}},
	}}
	router := gin.New()
	RegisterSignalRoutes(router.Group("/api/v1"), scanner)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/markets/signals?type=volume_surge&params=%7B%22window%22%3A20%7D", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if scanner.request.Type != "volume_surge" || scanner.request.Params != `{"window":20}` {
		t.Fatalf("request = %#v, want decoded query", scanner.request)
	}
	var body MarketSignals
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode signal response: %v", err)
	}
	if body.AsOf != "2024-01-21" || len(body.Signals) != 1 || body.Signals[0].Code != "A" {
		t.Fatalf("body = %#v, want signal result", body)
	}
}

func TestRegisterSignalRoutesMapsValidationAndHistoryErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
		code       api.ErrorCode
	}{
		{name: "validation", err: &SignalValidationError{Fields: map[string]string{"params.window": "必须是 20、60 或 120"}}, statusCode: http.StatusBadRequest, code: api.CodeValidation},
		{name: "history", err: &SignalHistoryError{InstrumentCode: "A", Required: 21, Available: 2}, statusCode: http.StatusUnprocessableEntity, code: api.CodeInsufficientHistory},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			RegisterSignalRoutes(router.Group("/api/v1"), &fakeSignalScanner{err: test.err})
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/v1/markets/signals?type=new_high", nil)
			router.ServeHTTP(response, request)

			if response.Code != test.statusCode {
				t.Fatalf("status = %d, want %d", response.Code, test.statusCode)
			}
			var body api.ErrorResponse
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if body.Code != test.code || body.Message == "" {
				t.Fatalf("body = %#v, want code %q", body, test.code)
			}
		})
	}
}

func TestRegisterSignalRoutesReturnsSafeDependencyError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterSignalRoutes(router.Group("/api/v1"), &fakeSignalScanner{err: errors.New("secret database detail")})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/markets/signals?type=new_high", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable || strings.Contains(response.Body.String(), "secret database detail") {
		t.Fatalf("response = %q, want safe dependency error", response.Body.String())
	}
}
