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

type fakeOverviewQuery struct {
	overview StockOverview
	err      error
}

func (query fakeOverviewQuery) Overview(context.Context, string) (StockOverview, error) {
	return query.overview, query.err
}

func TestRegisterOverviewRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterOverviewRoutes(router.Group("/api/v1"), fakeOverviewQuery{overview: StockOverview{
		Symbol: "000001.SZ", Name: "平安银行", Industry: "银行",
		Quote:     QuoteOverview{Last: "10.31", AsOf: "2024-06-28"},
		Metrics:   MetricsOverview{PETTM: MetricOverview{Value: stockStringPointer("5.82"), AsOf: stockStringPointer("2024-06-28"), Basis: stockStringPointer("ttm")}},
		Sparkline: SparklineOverview{Period: "20d", Points: []SparklinePoint{{TradeDate: "2024-06-28", Open: "10.20", High: "10.40", Low: "10.10", Close: "10.31"}}},
	}})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/stocks/000001.SZ", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"symbol":"000001.SZ"`) || !strings.Contains(response.Body.String(), `"open":"10.20"`) {
		t.Fatalf("status/body = %d/%s, want stock overview", response.Code, response.Body.String())
	}
}

func stockStringPointer(value string) *string {
	return &value
}

func TestRegisterOverviewRoutesMapsErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		err        error
		path       string
		statusCode int
		code       api.ErrorCode
	}{
		{name: "invalid symbol", err: &SymbolValidationError{Message: "invalid"}, path: "/api/v1/stocks/bad", statusCode: http.StatusBadRequest, code: api.CodeValidation},
		{name: "unknown instrument", err: ErrInstrumentNotFound, path: "/api/v1/stocks/999999.SZ", statusCode: http.StatusNotFound, code: api.CodeNotFound},
		{name: "dependency failure", err: errors.New("secret database detail"), path: "/api/v1/stocks/000001.SZ", statusCode: http.StatusServiceUnavailable, code: api.CodeDependencyUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			RegisterOverviewRoutes(router.Group("/api/v1"), fakeOverviewQuery{err: test.err})
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
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
