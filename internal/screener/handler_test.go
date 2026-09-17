package screener

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/disturb-yy/stock-quant/internal/screener/domain"
	"github.com/disturb-yy/stock-quant/pkg/api"
	"github.com/gin-gonic/gin"
)

type fakeRunQuery struct {
	response ScreenerRunResponse
	err      error
}

func (query fakeRunQuery) Run(context.Context, ScreenerRunRequest) (ScreenerRunResponse, error) {
	return query.response, query.err
}

func TestRegisterRoutesRun(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), fakeRunQuery{response: ScreenerRunResponse{MatchedCount: 0, ReturnedCount: 0, Results: []domain.ResultRow{}}})
	requestBody := `{"spec":{"universe_id":"cn_a_share_active","filters":[],"ranking":{"field_id":"technical.close","direction":"desc"},"top_n":10}}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/screeners/run", strings.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestRegisterRoutesRejectsMalformedBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), fakeRunQuery{})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/screeners/run", strings.NewReader(`{"spec":`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
	var body api.ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.Code != api.CodeValidation {
		t.Fatalf("code = %q, want %q", body.Code, api.CodeValidation)
	}
}

func TestRegisterRoutesMapsUniverseNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), fakeRunQuery{err: domain.ErrUniverseNotFound})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/screeners/run", strings.NewReader(`{}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", response.Code)
	}
}

func TestRegisterRoutesWithServiceMapsValidationAndEmptyResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	value := "10"
	service, err := NewService(&fakeSnapshotReader{input: domain.ExecutionInput{
		Universe: domain.Universe{ID: domain.ActiveAShareUniverse, Name: domain.ActiveAShareName},
		Eligible: []domain.Candidate{{Symbol: "A.SH", Name: "甲", Values: map[string]domain.Observation{"technical.close": {Value: &value}}}},
		Snapshot: domain.Snapshot{AsOf: "2024-06-28", FieldAsOf: map[string]string{"technical.close": "2024-06-28"}, DefinitionVersions: map[string]string{}},
		Source:   domain.Source{Mode: "demo", Provider: "mysql-demo-fixture", SeedVersion: "v1", AsOf: "2024-06-28"},
	}})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), service)
	requestBody := `{"spec":{"universe_id":"cn_a_share_active","filters":[{"field_id":"technical.close","operator":"gt","value":"999"}],"ranking":{"field_id":"technical.close","direction":"desc"},"top_n":1}}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/screeners/run", strings.NewReader(requestBody))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"matched_count":0`) {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	invalidRequest := httptest.NewRequest(http.MethodPost, "/api/v1/screeners/run", strings.NewReader(`{"spec":{"universe_id":"cn_a_share_active","ranking":{"field_id":"technical.close","direction":"desc"},"top_n":0}}`))
	invalidResponse := httptest.NewRecorder()
	router.ServeHTTP(invalidResponse, invalidRequest)
	if invalidResponse.Code != http.StatusBadRequest {
		t.Fatalf("invalid status = %d, want 400", invalidResponse.Code)
	}
}
