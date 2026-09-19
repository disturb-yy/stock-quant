package pool

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/disturb-yy/stock-quant/internal/pool/domain"
	"github.com/disturb-yy/stock-quant/pkg/api"
	"github.com/gin-gonic/gin"
)

type fakeStockPoolQuery struct {
	created      domain.StockPool
	list         StockPoolListResponse
	members      StockPoolMemberListResponse
	added        StockPoolMemberAddResponse
	deleted      StockPoolMemberDeleteResponse
	createErr    error
	listErr      error
	getErr       error
	summary      domain.StockPoolSummary
	summaryErr   error
	membersErr   error
	addErr       error
	deleteErr    error
	input        domain.StockPoolInput
	listReq      StockPoolListRequest
	membersReq   StockPoolMemberListRequest
	memberID     int64
	memberSymbol string
}

func (query *fakeStockPoolQuery) Create(_ context.Context, input domain.StockPoolInput) (domain.StockPool, error) {
	query.input = input
	return query.created, query.createErr
}

func (query *fakeStockPoolQuery) List(_ context.Context, request StockPoolListRequest) (StockPoolListResponse, error) {
	query.listReq = request
	return query.list, query.listErr
}

func (query *fakeStockPoolQuery) Get(context.Context, int64) (domain.StockPool, error) {
	return query.created, query.getErr
}

func (query *fakeStockPoolQuery) Summary(context.Context, int64) (domain.StockPoolSummary, error) {
	return query.summary, query.summaryErr
}

func (query *fakeStockPoolQuery) ListMembers(_ context.Context, id int64, request StockPoolMemberListRequest) (StockPoolMemberListResponse, error) {
	query.memberID, query.membersReq = id, request
	return query.members, query.membersErr
}

func (query *fakeStockPoolQuery) AddMember(_ context.Context, id int64, symbol string) (StockPoolMemberAddResponse, error) {
	query.memberID, query.memberSymbol = id, symbol
	return query.added, query.addErr
}

func (query *fakeStockPoolQuery) DeleteMember(_ context.Context, id int64, symbol string) (StockPoolMemberDeleteResponse, error) {
	query.memberID, query.memberSymbol = id, symbol
	return query.deleted, query.deleteErr
}

func TestRegisterRoutesSupportsCreateListAndDetail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	query := &fakeStockPoolQuery{created: testStockPool(1), list: StockPoolListResponse{Data: []domain.StockPool{testStockPool(1)}}, summary: testStockPoolSummary(1)}
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), query)

	createResponse := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/stock-pools", strings.NewReader(`{"name":"红利观察","description":null}`))
	router.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusOK || !strings.Contains(createResponse.Body.String(), `"source":"manual"`) || !strings.Contains(createResponse.Body.String(), `"member_count":0`) {
		t.Fatalf("create status = %d, body = %s", createResponse.Code, createResponse.Body.String())
	}
	if query.input.Name != "红利观察" || query.input.Description != nil {
		t.Fatalf("create input = %#v, want approved fields", query.input)
	}

	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, httptest.NewRequest(http.MethodGet, "/api/v1/stock-pools?q=%E7%BA%A2%E5%88%A9&page=2&page_size=10", nil))
	if listResponse.Code != http.StatusOK || query.listReq.Search != "红利" || query.listReq.Page != 2 || query.listReq.PageSize != 10 {
		t.Fatalf("list status = %d, request = %#v", listResponse.Code, query.listReq)
	}

	detailResponse := httptest.NewRecorder()
	router.ServeHTTP(detailResponse, httptest.NewRequest(http.MethodGet, "/api/v1/stock-pools/1", nil))
	if detailResponse.Code != http.StatusOK || !strings.Contains(detailResponse.Body.String(), `"id":1`) {
		t.Fatalf("detail status = %d, body = %s", detailResponse.Code, detailResponse.Body.String())
	}

	summaryResponse := httptest.NewRecorder()
	router.ServeHTTP(summaryResponse, httptest.NewRequest(http.MethodGet, "/api/v1/stock-pools/1/summary", nil))
	if summaryResponse.Code != http.StatusOK || !strings.Contains(summaryResponse.Body.String(), `"type":"manual"`) {
		t.Fatalf("summary status = %d, body = %s", summaryResponse.Code, summaryResponse.Body.String())
	}

	membersResponse := httptest.NewRecorder()
	router.ServeHTTP(membersResponse, httptest.NewRequest(http.MethodGet, "/api/v1/stock-pools/1/members?page=2&page_size=10", nil))
	if membersResponse.Code != http.StatusOK || query.memberID != 1 || query.membersReq.Page != 2 || query.membersReq.PageSize != 10 {
		t.Fatalf("members status = %d, request = %#v", membersResponse.Code, query.membersReq)
	}

	addResponse := httptest.NewRecorder()
	router.ServeHTTP(addResponse, httptest.NewRequest(http.MethodPost, "/api/v1/stock-pools/1/members", strings.NewReader(`{"symbol":"000001.SZ"}`)))
	if addResponse.Code != http.StatusOK || query.memberSymbol != "000001.SZ" {
		t.Fatalf("add member status = %d, body = %s", addResponse.Code, addResponse.Body.String())
	}

	deleteResponse := httptest.NewRecorder()
	router.ServeHTTP(deleteResponse, httptest.NewRequest(http.MethodDelete, "/api/v1/stock-pools/1/members/000001.SZ", nil))
	if deleteResponse.Code != http.StatusOK || query.memberSymbol != "000001.SZ" {
		t.Fatalf("delete member status = %d, body = %s", deleteResponse.Code, deleteResponse.Body.String())
	}
}

func TestRegisterRoutesRejectsServerOwnedFieldsAndMapsErrors(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		query      *fakeStockPoolQuery
		statusCode int
		code       api.ErrorCode
	}{
		{name: "server owned source", method: http.MethodPost, path: "/api/v1/stock-pools", body: `{"name":"池","source":"screener"}`, query: &fakeStockPoolQuery{}, statusCode: http.StatusBadRequest, code: api.CodeValidation},
		{name: "server owned members", method: http.MethodPost, path: "/api/v1/stock-pools", body: `{"name":"池","member_count":1}`, query: &fakeStockPoolQuery{}, statusCode: http.StatusBadRequest, code: api.CodeValidation},
		{name: "invalid pagination", method: http.MethodGet, path: "/api/v1/stock-pools?page=0", query: &fakeStockPoolQuery{}, statusCode: http.StatusBadRequest, code: api.CodeInvalidPagination},
		{name: "invalid id", method: http.MethodGet, path: "/api/v1/stock-pools/nope", query: &fakeStockPoolQuery{}, statusCode: http.StatusBadRequest, code: api.CodeValidation},
		{name: "not found", method: http.MethodGet, path: "/api/v1/stock-pools/1", query: &fakeStockPoolQuery{getErr: domain.ErrStockPoolNotFound}, statusCode: http.StatusNotFound, code: api.CodeNotFound},
		{name: "dependency", method: http.MethodGet, path: "/api/v1/stock-pools/1", query: &fakeStockPoolQuery{getErr: errors.New("database unavailable")}, statusCode: http.StatusServiceUnavailable, code: api.CodeDependencyUnavailable},
		{name: "summary not found", method: http.MethodGet, path: "/api/v1/stock-pools/1/summary", query: &fakeStockPoolQuery{summaryErr: domain.ErrStockPoolNotFound}, statusCode: http.StatusNotFound, code: api.CodeNotFound},
		{name: "summary dependency", method: http.MethodGet, path: "/api/v1/stock-pools/1/summary", query: &fakeStockPoolQuery{summaryErr: errors.New("database unavailable")}, statusCode: http.StatusServiceUnavailable, code: api.CodeDependencyUnavailable},
		{name: "member conflict", method: http.MethodPost, path: "/api/v1/stock-pools/1/members", body: `{"symbol":"000001.SZ"}`, query: &fakeStockPoolQuery{addErr: domain.ErrStockPoolMemberConflict}, statusCode: http.StatusConflict, code: api.CodeConflict},
		{name: "member missing", method: http.MethodDelete, path: "/api/v1/stock-pools/1/members/000001.SZ", query: &fakeStockPoolQuery{deleteErr: domain.ErrStockPoolMemberNotFound}, statusCode: http.StatusNotFound, code: api.CodeNotFound},
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

func testStockPool(id int64) domain.StockPool {
	return domain.StockPool{ID: id, Name: "红利观察", Source: domain.SourceManual, MemberCount: 0, CreatedAt: time.Unix(0, 0).UTC(), UpdatedAt: time.Unix(0, 0).UTC()}
}

func testStockPoolSummary(id int64) domain.StockPoolSummary {
	return domain.StockPoolSummary{ID: id, Name: "红利观察", Source: domain.StockPoolSourceSummary{Type: domain.SourceManual}}
}
