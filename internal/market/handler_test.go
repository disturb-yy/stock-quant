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

type fakeOverviewQuery struct {
	overview MarketOverview
	err      error
}

func (query fakeOverviewQuery) Overview(context.Context) (MarketOverview, error) {
	return query.overview, query.err
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
