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

type fakeRankingQuery struct {
	result  MarketRankings
	err     error
	request RankingRequest
}

func (query *fakeRankingQuery) Rank(_ context.Context, request RankingRequest) (MarketRankings, error) {
	query.request = request
	return query.result, query.err
}

func TestRegisterRankingRoutesReturnsRankingsAndParsesPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	query := &fakeRankingQuery{result: MarketRankings{
		Metric: "gain", AsOf: "2024-06-28",
		Data: []RankingStock{{Rank: 1, Code: "300750.SZ", Name: "宁德时代", Value: "2.40"}},
	}}
	router := gin.New()
	RegisterRankingRoutes(router.Group("/api/v1"), query)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/markets/rankings?metric=gain&page=2&page_size=10", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if query.request.Metric != "gain" || query.request.Page != 2 || query.request.PageSize != 10 {
		t.Fatalf("request = %#v, want parsed ranking request", query.request)
	}
	if !strings.Contains(response.Body.String(), `"code":"300750.SZ"`) {
		t.Fatalf("response = %q, want ranking result", response.Body.String())
	}
}

func TestRegisterRankingRoutesMapsValidationPaginationHistoryAndDependencyErrors(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		err        error
		statusCode int
		code       api.ErrorCode
	}{
		{name: "missing metric", path: "/api/v1/markets/rankings", statusCode: http.StatusBadRequest, code: api.CodeValidation},
		{name: "invalid pagination", path: "/api/v1/markets/rankings?metric=gain&page=0", statusCode: http.StatusBadRequest, code: api.CodeInvalidPagination},
		{name: "invalid metric from service", path: "/api/v1/markets/rankings?metric=gain", err: &RankingValidationError{Fields: map[string]string{"metric": "invalid"}}, statusCode: http.StatusBadRequest, code: api.CodeValidation},
		{name: "insufficient history", path: "/api/v1/markets/rankings?metric=gain", err: &RankingHistoryError{InstrumentCode: "A", RequiredBars: 2, AvailableBars: 1}, statusCode: http.StatusUnprocessableEntity, code: api.CodeInsufficientHistory},
		{name: "no data", path: "/api/v1/markets/rankings?metric=gain", err: ErrRankingNoData, statusCode: http.StatusServiceUnavailable, code: api.CodeDependencyUnavailable},
		{name: "dependency", path: "/api/v1/markets/rankings?metric=gain", err: errors.New("secret database detail"), statusCode: http.StatusServiceUnavailable, code: api.CodeDependencyUnavailable},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			RegisterRankingRoutes(router.Group("/api/v1"), &fakeRankingQuery{err: test.err})
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))

			if response.Code != test.statusCode {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.statusCode, response.Body.String())
			}
			var body api.ErrorResponse
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if body.Code != test.code || strings.Contains(response.Body.String(), "secret database detail") {
				t.Fatalf("body = %#v, want code %q and safe error", body, test.code)
			}
		})
	}
}
