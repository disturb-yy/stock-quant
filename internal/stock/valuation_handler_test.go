package stock

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/disturb-yy/stock-quant/pkg/api"
	"github.com/gin-gonic/gin"
)

type fakeValuationQuery struct {
	valuation StockValuation
	err       error
}

func (query fakeValuationQuery) Valuation(context.Context, ValuationRequest) (StockValuation, error) {
	return query.valuation, query.err
}

func TestRegisterValuationRoutesReturnsSuccessAndRejectsDuplicateRange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterValuationRoutes(router.Group("/api/v1"), fakeValuationQuery{valuation: StockValuation{Symbol: "000001.SZ", RequestedRange: ValuationRangeFiveYears, Metrics: ValuationMetrics{PETTM: ValuationMetric{History: []ValuationHistoryPoint{}}}}})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/stocks/000001.SZ/valuation?range=5y", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"requested_range":"5y"`) || !strings.Contains(response.Body.String(), `"history":[]`) {
		t.Fatalf("success status/body = %d/%s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/stocks/000001.SZ/valuation?range=3y&range=5y", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("duplicate range status = %d, want 400", response.Code)
	}
	var body api.ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode validation response: %v", err)
	}
	if body.Code != api.CodeValidation {
		t.Fatalf("validation code = %q, want %q", body.Code, api.CodeValidation)
	}
}

func TestRegisterValuationRoutesMapsErrorsSafely(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		err        error
		statusCode int
		code       api.ErrorCode
	}{
		{name: "unknown stock", err: ErrValuationInstrumentNotFound, statusCode: http.StatusNotFound, code: api.CodeNotFound},
		{name: "dependency", err: errors.New("secret database detail"), statusCode: http.StatusServiceUnavailable, code: api.CodeDependencyUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			RegisterValuationRoutes(router.Group("/api/v1"), fakeValuationQuery{err: test.err})
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/stocks/999999.SZ/valuation", nil))
			if response.Code != test.statusCode {
				t.Fatalf("status = %d, want %d", response.Code, test.statusCode)
			}
			var body api.ErrorResponse
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if body.Code != test.code || strings.Contains(response.Body.String(), "secret database detail") {
				t.Fatalf("error response = %#v, want safe %q", body, test.code)
			}
		})
	}
}
