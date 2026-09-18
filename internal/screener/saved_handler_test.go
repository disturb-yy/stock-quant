package screener

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/disturb-yy/stock-quant/internal/screener/domain"
	"github.com/disturb-yy/stock-quant/pkg/api"
	"github.com/gin-gonic/gin"
)

type fakeSavedScreenerQuery struct {
	created   domain.SavedScreener
	list      SavedScreenerListResponse
	createErr error
	listErr   error
	getErr    error
	updateErr error
	listReq   SavedScreenerListRequest
	updateVer int64
}

func (query *fakeSavedScreenerQuery) Create(_ context.Context, input domain.SavedScreenerInput) (domain.SavedScreener, error) {
	if query.createErr != nil {
		return domain.SavedScreener{}, query.createErr
	}
	query.created.Name = input.Name
	query.created.Spec = input.Spec
	return query.created, nil
}

func (query *fakeSavedScreenerQuery) List(_ context.Context, request SavedScreenerListRequest) (SavedScreenerListResponse, error) {
	query.listReq = request
	return query.list, query.listErr
}

func (query *fakeSavedScreenerQuery) Get(context.Context, int64) (domain.SavedScreener, error) {
	return query.created, query.getErr
}

func (query *fakeSavedScreenerQuery) Update(_ context.Context, _ int64, version int64, _ domain.SavedScreenerInput) (domain.SavedScreener, error) {
	query.updateVer = version
	return query.created, query.updateErr
}

func TestRegisterSavedRoutesSupportsCRUDAndPreservesRunRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	query := &fakeSavedScreenerQuery{created: domain.SavedScreener{ID: 1, Version: 1}, list: SavedScreenerListResponse{Data: []domain.SavedScreener{}}}
	router := gin.New()
	group := router.Group("/api/v1")
	RegisterRoutes(group, fakeRunQuery{response: ScreenerRunResponse{Results: []domain.ResultRow{}}})
	RegisterSavedRoutes(group, query)

	createResponse := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/screeners", strings.NewReader(`{"name":"方案","description":null,"spec":{"universe_id":"cn_a_share_active","filters":[],"ranking":{"field_id":"technical.close","direction":"desc"},"top_n":1}}`))
	router.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusOK || !strings.Contains(createResponse.Body.String(), `"id":1`) {
		t.Fatalf("create status = %d, body = %s", createResponse.Code, createResponse.Body.String())
	}

	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, httptest.NewRequest(http.MethodGet, "/api/v1/screeners?page=2&page_size=10", nil))
	if listResponse.Code != http.StatusOK || query.listReq.Page != 2 || query.listReq.PageSize != 10 {
		t.Fatalf("list status = %d, request = %#v", listResponse.Code, query.listReq)
	}

	runResponse := httptest.NewRecorder()
	router.ServeHTTP(runResponse, httptest.NewRequest(http.MethodPost, "/api/v1/screeners/run", strings.NewReader(`{}`)))
	if runResponse.Code != http.StatusOK {
		t.Fatalf("run route status = %d, body = %s", runResponse.Code, runResponse.Body.String())
	}
	methodResponse := httptest.NewRecorder()
	router.ServeHTTP(methodResponse, httptest.NewRequest(http.MethodGet, "/api/v1/screeners/run", nil))
	if methodResponse.Code != http.StatusMethodNotAllowed {
		t.Fatalf("run method status = %d, want %d", methodResponse.Code, http.StatusMethodNotAllowed)
	}

	updateResponse := httptest.NewRecorder()
	updateBody := `{"name":"更新方案","description":null,"spec":{"universe_id":"cn_a_share_active","filters":[],"ranking":{"field_id":"technical.close","direction":"desc"},"top_n":1},"version":3}`
	router.ServeHTTP(updateResponse, httptest.NewRequest(http.MethodPut, "/api/v1/screeners/1", strings.NewReader(updateBody)))
	if updateResponse.Code != http.StatusOK || query.updateVer != 3 {
		t.Fatalf("update status = %d, version = %d", updateResponse.Code, query.updateVer)
	}
}

func TestRegisterSavedRoutesMapsValidationNotFoundConflictAndDependency(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		query      *fakeSavedScreenerQuery
		statusCode int
		code       api.ErrorCode
	}{
		{name: "invalid id", method: http.MethodGet, path: "/api/v1/screeners/nope", query: &fakeSavedScreenerQuery{}, statusCode: http.StatusBadRequest, code: api.CodeValidation},
		{name: "unknown field", method: http.MethodPost, path: "/api/v1/screeners", body: `{"name":"方案","unexpected":true}`, query: &fakeSavedScreenerQuery{}, statusCode: http.StatusBadRequest, code: api.CodeValidation},
		{name: "not found", method: http.MethodGet, path: "/api/v1/screeners/1", query: &fakeSavedScreenerQuery{getErr: domain.ErrSavedScreenerNotFound}, statusCode: http.StatusNotFound, code: api.CodeNotFound},
		{name: "conflict", method: http.MethodPut, path: "/api/v1/screeners/1", body: `{"name":"方案","description":null,"spec":{},"version":1}`, query: &fakeSavedScreenerQuery{updateErr: &domain.SavedScreenerVersionConflictError{CurrentVersion: 2}}, statusCode: http.StatusConflict, code: api.CodeConflict},
		{name: "dependency", method: http.MethodGet, path: "/api/v1/screeners/1", query: &fakeSavedScreenerQuery{getErr: errors.New("database unavailable")}, statusCode: http.StatusServiceUnavailable, code: api.CodeDependencyUnavailable},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			RegisterSavedRoutes(router.Group("/api/v1"), test.query)
			response := httptest.NewRecorder()
			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			router.ServeHTTP(response, request)
			if response.Code != test.statusCode {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.statusCode, response.Body.String())
			}
			var body api.ErrorResponse
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if body.Code != test.code {
				t.Fatalf("code = %q, want %q", body.Code, test.code)
			}
		})
	}
}
