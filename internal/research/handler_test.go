package research

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/disturb-yy/stock-quant/internal/research/domain"
	"github.com/disturb-yy/stock-quant/pkg/api"
	"github.com/gin-gonic/gin"
)

type fakeResearchQuery struct {
	created   domain.ResearchProject
	list      ResearchListResponse
	createErr error
	listErr   error
	getErr    error
	listReq   ResearchListRequest
}

func (query *fakeResearchQuery) Create(_ context.Context, _ domain.ResearchProjectInput) (domain.ResearchProject, error) {
	return query.created, query.createErr
}

func (query *fakeResearchQuery) List(_ context.Context, request ResearchListRequest) (ResearchListResponse, error) {
	query.listReq = request
	return query.list, query.listErr
}

func (query *fakeResearchQuery) Get(_ context.Context, _ int64) (domain.ResearchProject, error) {
	return query.created, query.getErr
}

func TestRegisterRoutesSupportsCreateListAndDetail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	query := &fakeResearchQuery{
		created: domain.ResearchProject{ID: 1, Name: "银行研究", CreatedAt: time.Unix(0, 0).UTC()},
		list:    ResearchListResponse{Data: []domain.ResearchProject{}},
	}
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), query)

	createResponse := httptest.NewRecorder()
	router.ServeHTTP(createResponse, httptest.NewRequest(http.MethodPost, "/api/v1/research", strings.NewReader(`{"name":"银行研究","description":null}`)))
	if createResponse.Code != http.StatusOK || !strings.Contains(createResponse.Body.String(), `"id":1`) {
		t.Fatalf("create status = %d, body = %s", createResponse.Code, createResponse.Body.String())
	}

	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, httptest.NewRequest(http.MethodGet, "/api/v1/research?page=2&page_size=10", nil))
	if listResponse.Code != http.StatusOK || query.listReq.Page != 2 || query.listReq.PageSize != 10 {
		t.Fatalf("list status = %d, request = %#v", listResponse.Code, query.listReq)
	}
	if strings.Contains(listResponse.Body.String(), `"data":null`) {
		t.Fatal("empty research list must serialize data as an array")
	}

	detailResponse := httptest.NewRecorder()
	router.ServeHTTP(detailResponse, httptest.NewRequest(http.MethodGet, "/api/v1/research/1", nil))
	if detailResponse.Code != http.StatusOK || !strings.Contains(detailResponse.Body.String(), `"id":1`) {
		t.Fatalf("detail status = %d, body = %s", detailResponse.Code, detailResponse.Body.String())
	}
}

func TestRegisterRoutesMapsValidationNotFoundAndDependency(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		query      *fakeResearchQuery
		statusCode int
		code       api.ErrorCode
	}{
		{name: "invalid body", method: http.MethodPost, path: "/api/v1/research", body: `{"unknown":true}`, query: &fakeResearchQuery{}, statusCode: http.StatusBadRequest, code: api.CodeValidation},
		{name: "invalid pagination", method: http.MethodGet, path: "/api/v1/research?page=0", query: &fakeResearchQuery{}, statusCode: http.StatusBadRequest, code: api.CodeInvalidPagination},
		{name: "invalid id", method: http.MethodGet, path: "/api/v1/research/nope", query: &fakeResearchQuery{}, statusCode: http.StatusBadRequest, code: api.CodeValidation},
		{name: "not found", method: http.MethodGet, path: "/api/v1/research/1", query: &fakeResearchQuery{getErr: domain.ErrResearchProjectNotFound}, statusCode: http.StatusNotFound, code: api.CodeNotFound},
		{name: "dependency", method: http.MethodGet, path: "/api/v1/research/1", query: &fakeResearchQuery{getErr: errors.New("database unavailable")}, statusCode: http.StatusServiceUnavailable, code: api.CodeDependencyUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			RegisterRoutes(router.Group("/api/v1"), test.query)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(test.method, test.path, strings.NewReader(test.body)))
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
