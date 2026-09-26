package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/disturb-yy/stock-quant/internal/data/application"
	"github.com/disturb-yy/stock-quant/internal/data/domain"
	"github.com/gin-gonic/gin"
)

type fakeService struct {
	createdTask  domain.SyncTask
	listed       domain.TaskPage
	detail       domain.SyncTask
	retryTask    domain.SyncTask
	createErr    error
	listErr      error
	detailErr    error
	retryErr     error
	lastPage     int
	lastPageSize int
	lastInput    application.CreateTaskInput
}

func (service *fakeService) CreateTask(_ context.Context, input application.CreateTaskInput) (domain.SyncTask, error) {
	service.lastInput = input
	return service.createdTask, service.createErr
}

func (service *fakeService) GetTask(_ context.Context, _ string) (domain.SyncTask, error) {
	return service.detail, service.detailErr
}

func (service *fakeService) ListTasks(_ context.Context, page, pageSize int) (domain.TaskPage, error) {
	service.lastPage = page
	service.lastPageSize = pageSize
	return service.listed, service.listErr
}

func (service *fakeService) RetryFailedTask(_ context.Context, _ string) (domain.SyncTask, error) {
	return service.retryTask, service.retryErr
}

func TestHandlerCreateSuccessAndValidation(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string
		wantTarget domain.SyncTarget
	}{
		{name: "success", body: `{"target":"daily_bars","start_date":"2026-09-01","end_date":"2026-09-02"}`, wantStatus: http.StatusAccepted, wantTarget: domain.TargetDailyBars},
		{name: "invalid target", body: `{"target":"unknown"}`, wantStatus: http.StatusBadRequest, wantCode: "INVALID_REQUEST"},
		{name: "invalid date", body: `{"target":"daily_bars","start_date":"2026-09-02","end_date":"2026-09-01"}`, wantStatus: http.StatusBadRequest, wantCode: "INVALID_REQUEST"},
		{name: "basic info rejects dates", body: `{"target":"basic_info","start_date":"2026-09-01","end_date":"2026-09-02"}`, wantStatus: http.StatusBadRequest, wantCode: "INVALID_REQUEST"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &fakeService{createdTask: sampleTask("task-create", domain.StatusPending)}
			recorder := performRequest(newRouterForTest(service), http.MethodPost, "/api/v1/stock/data/sync-tasks", test.body)
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, test.wantStatus, recorder.Body.String())
			}
			if test.wantCode != "" {
				assertErrorCode(t, recorder, test.wantCode)
				return
			}
			if service.lastInput.Target != test.wantTarget || service.lastInput.Trigger != domain.TriggerManual {
				t.Fatalf("unexpected input: %#v", service.lastInput)
			}
			var response taskResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.TaskID != "task-create" || response.Result != nil || response.StartedAt != nil {
				t.Fatalf("unexpected response: %#v", response)
			}
		})
	}
}

func TestHandlerListPagination(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantPage   int
		wantSize   int
		wantCode   string
	}{
		{name: "defaults", wantStatus: http.StatusOK, wantPage: 1, wantSize: 10},
		{name: "maximum page size", query: "?page=2&page_size=50", wantStatus: http.StatusOK, wantPage: 2, wantSize: 50},
		{name: "page starts at one", query: "?page=0", wantStatus: http.StatusBadRequest, wantCode: "INVALID_PAGINATION"},
		{name: "page size maximum", query: "?page_size=51", wantStatus: http.StatusBadRequest, wantCode: "INVALID_PAGINATION"},
		{name: "not a number", query: "?page_size=nope", wantStatus: http.StatusBadRequest, wantCode: "INVALID_PAGINATION"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &fakeService{listed: domain.TaskPage{Items: []domain.SyncTask{}, Page: test.wantPage, PageSize: test.wantSize}}
			recorder := performRequest(newRouterForTest(service), http.MethodGet, "/api/v1/stock/data/sync-tasks"+test.query, "")
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, test.wantStatus, recorder.Body.String())
			}
			if test.wantCode != "" {
				assertErrorCode(t, recorder, test.wantCode)
				return
			}
			if service.lastPage != test.wantPage || service.lastPageSize != test.wantSize {
				t.Fatalf("pagination = (%d,%d), want (%d,%d)", service.lastPage, service.lastPageSize, test.wantPage, test.wantSize)
			}
			if !strings.Contains(recorder.Body.String(), `"items":[]`) {
				t.Fatalf("empty items must be an array: %s", recorder.Body.String())
			}
		})
	}
}

func TestHandlerDetailAndRetryErrors(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		service    *fakeService
		wantStatus int
		wantCode   string
	}{
		{name: "detail success", method: http.MethodGet, path: "/api/v1/stock/data/sync-tasks/task-1", service: &fakeService{detail: sampleTask("task-1", domain.StatusSucceeded)}, wantStatus: http.StatusOK},
		{name: "detail not found", method: http.MethodGet, path: "/api/v1/stock/data/sync-tasks/missing", service: &fakeService{detailErr: domain.ErrTaskNotFound}, wantStatus: http.StatusNotFound, wantCode: "SYNC_TASK_NOT_FOUND"},
		{name: "retry success", method: http.MethodPost, path: "/api/v1/stock/data/sync-tasks/task-1/retry", service: &fakeService{retryTask: sampleTask("task-2", domain.StatusPending)}, wantStatus: http.StatusAccepted},
		{name: "retry not retryable", method: http.MethodPost, path: "/api/v1/stock/data/sync-tasks/task-1/retry", service: &fakeService{retryErr: domain.ErrTaskNotRetryable}, wantStatus: http.StatusConflict, wantCode: "SYNC_TASK_NOT_RETRYABLE"},
		{name: "retry not found", method: http.MethodPost, path: "/api/v1/stock/data/sync-tasks/missing/retry", service: &fakeService{retryErr: domain.ErrTaskNotFound}, wantStatus: http.StatusNotFound, wantCode: "SYNC_TASK_NOT_FOUND"},
		{name: "retry conflict", method: http.MethodPost, path: "/api/v1/stock/data/sync-tasks/task-1/retry", service: &fakeService{retryErr: domain.ErrTaskConflict}, wantStatus: http.StatusConflict, wantCode: "SYNC_TASK_CONFLICT"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := performRequest(newRouterForTest(test.service), test.method, test.path, "")
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, test.wantStatus, recorder.Body.String())
			}
			if test.wantCode != "" {
				assertErrorCode(t, recorder, test.wantCode)
			}
		})
	}
}

func TestHandlerErrorJSONDoesNotExposeInternalError(t *testing.T) {
	service := &fakeService{createErr: errors.New("sql: DSN secret-token stack trace")}
	recorder := performRequest(newRouterForTest(service), http.MethodPost, "/api/v1/stock/data/sync-tasks", `{"target":"basic_info"}`)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), "secret-token") || strings.Contains(recorder.Body.String(), "stack trace") {
		t.Fatalf("internal details leaked: %s", recorder.Body.String())
	}
	assertErrorCode(t, recorder, "INTERNAL_SERVER_ERROR")
}

func TestHandlerMapsDataSourceUnavailable(t *testing.T) {
	service := &fakeService{createErr: domain.ErrDataSourceUnavailable}
	recorder := performRequest(newRouterForTest(service), http.MethodPost, "/api/v1/stock/data/sync-tasks", `{"target":"basic_info"}`)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", recorder.Code)
	}
	assertErrorCode(t, recorder, "DATA_SOURCE_UNAVAILABLE")
}

func newRouterForTest(service Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), service)
	return router
}

func performRequest(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	return recorder
}

func assertErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, want string) {
	t.Helper()
	var response errorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode error response: %v; body=%s", err, recorder.Body.String())
	}
	if response.Code != want || response.Message == "" {
		t.Fatalf("error response = %#v, want code %s", response, want)
	}
}

func sampleTask(id string, status domain.TaskStatus) domain.SyncTask {
	created := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	dataAsOf := end
	var result *domain.SyncResult
	if status == domain.StatusSucceeded {
		result = &domain.SyncResult{ProcessedCount: 2, CreatedCount: 2}
	}
	if status != domain.StatusSucceeded {
		dataAsOf = time.Time{}
	}
	var dataAsOfPointer *time.Time
	if !dataAsOf.IsZero() {
		dataAsOfPointer = &dataAsOf
	}
	return domain.SyncTask{ID: id, Target: domain.TargetDailyBars, Trigger: domain.TriggerManual, Status: status,
		Source: domain.SourceProvenance{Provider: "mock", Mode: "mock"}, DateRange: domain.DateRange{Start: &start, End: &end},
		CreatedAt: created, UpdatedAt: created, DataAsOf: dataAsOfPointer, Result: result, MaxRetries: 3}
}
