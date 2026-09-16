package market

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

type fakeBarsQuery struct {
	result StockBars
	err    error
	call   int
}

func (query *fakeBarsQuery) Bars(context.Context, BarsRequest) (StockBars, error) {
	query.call++
	return query.result, query.err
}

func TestRegisterBarsRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	query := &fakeBarsQuery{result: StockBars{Symbol: "000001.SZ", Name: "平安银行", Bars: []StockBar{}}}
	router := gin.New()
	RegisterBarsRoutes(router.Group("/api/v1"), query)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/stocks/000001.SZ/bars?range=20d&adjust=qfq", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"symbol":"000001.SZ"`) {
		t.Fatalf("status/body = %d/%s, want bars response", response.Code, response.Body.String())
	}
	if query.call != 1 {
		t.Fatalf("Bars calls = %d, want 1", query.call)
	}
}

func TestRegisterBarsRoutesRejectsInvalidQueryShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	query := &fakeBarsQuery{result: StockBars{Symbol: "000001.SZ"}}
	router := gin.New()
	RegisterBarsRoutes(router.Group("/api/v1"), query)

	for _, path := range []string{
		"/api/v1/stocks/000001.SZ/bars?range=20d&from=2024-06-01&to=2024-06-28",
		"/api/v1/stocks/000001.SZ/bars?from=2024-06-01",
		"/api/v1/stocks/000001.SZ/bars?timeframe=",
		"/api/v1/stocks/000001.SZ/bars?benchmark=000300.SH&benchmark=000300.SH",
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("path %q status = %d, want %d", path, response.Code, http.StatusBadRequest)
		}
		var body api.ErrorResponse
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			t.Fatalf("decode %q error response: %v", path, err)
		}
		if body.Code != api.CodeValidation || body.Details == nil {
			t.Fatalf("path %q error = %#v, want validation details", path, body)
		}
	}
	if query.call != 0 {
		t.Fatalf("Bars calls = %d, want no call for invalid query", query.call)
	}
}

func TestRegisterBarsRoutesMapsErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		err        error
		statusCode int
		code       api.ErrorCode
	}{
		{name: "unknown instrument", err: ErrBarsInstrumentNotFound, statusCode: http.StatusNotFound, code: api.CodeNotFound},
		{name: "dependency failure", err: errors.New("secret database detail"), statusCode: http.StatusServiceUnavailable, code: api.CodeDependencyUnavailable},
		{name: "validation failure", err: &BarsValidationError{Fields: map[string]string{"adjust": "invalid"}}, statusCode: http.StatusBadRequest, code: api.CodeValidation},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			RegisterBarsRoutes(router.Group("/api/v1"), &fakeBarsQuery{err: test.err})
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/stocks/000001.SZ/bars", nil))
			if response.Code != test.statusCode {
				t.Fatalf("status = %d, want %d", response.Code, test.statusCode)
			}
			var body api.ErrorResponse
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if body.Code != test.code || strings.Contains(response.Body.String(), "secret database detail") {
				t.Fatalf("error response = %#v, want code %q without internal detail", body, test.code)
			}
		})
	}
}
