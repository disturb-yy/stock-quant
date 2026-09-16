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

type fakeFinancialsQuery struct {
	result  StockFinancials
	err     error
	call    int
	request FinancialsRequest
}

func (query *fakeFinancialsQuery) Financials(_ context.Context, request FinancialsRequest) (StockFinancials, error) {
	query.call++
	query.request = request
	return query.result, query.err
}

func TestRegisterFinancialsRoutesParsesQueryAndReturnsResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	query := &fakeFinancialsQuery{result: StockFinancials{Symbol: "000001.SZ", Reports: []FinancialReportView{}}}
	router := gin.New()
	RegisterFinancialsRoutes(router.Group("/api/v1"), query)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/stocks/000001.SZ/financials?period=quarterly&range=3y", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"symbol":"000001.SZ"`) {
		t.Fatalf("status/body = %d/%s, want financials response", response.Code, response.Body.String())
	}
	if query.call != 1 || query.request.Symbol != "000001.SZ" || query.request.Period != FinancialPeriodQuarterly || query.request.Range != FinancialRangeThreeYears {
		t.Fatalf("query call/request = %d/%#v, want one normalized query", query.call, query.request)
	}
}

func TestRegisterFinancialsRoutesRejectsInvalidQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	query := &fakeFinancialsQuery{}
	router := gin.New()
	RegisterFinancialsRoutes(router.Group("/api/v1"), query)

	for _, path := range []string{
		"/api/v1/stocks/000001.SZ/financials?period=monthly",
		"/api/v1/stocks/000001.SZ/financials?range=10y",
		"/api/v1/stocks/000001.SZ/financials?period=annual&period=quarterly",
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
		t.Fatalf("Financials calls = %d, want no call for invalid query", query.call)
	}
}

func TestRegisterFinancialsRoutesMapsErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		err        error
		statusCode int
		code       api.ErrorCode
	}{
		{name: "unknown instrument", err: ErrFinancialsInstrumentNotFound, statusCode: http.StatusNotFound, code: api.CodeNotFound},
		{name: "dependency failure", err: errors.New("secret database detail"), statusCode: http.StatusServiceUnavailable, code: api.CodeDependencyUnavailable},
		{name: "validation failure", err: &FinancialsValidationError{Fields: map[string]string{"period": "invalid"}}, statusCode: http.StatusBadRequest, code: api.CodeValidation},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			RegisterFinancialsRoutes(router.Group("/api/v1"), &fakeFinancialsQuery{err: test.err})
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/stocks/000001.SZ/financials", nil))
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
