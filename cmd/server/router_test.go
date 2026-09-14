package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/stock-ddd/pkg/api"
	"example.com/stock-ddd/pkg/config"
	"github.com/gin-gonic/gin"
)

func TestNewRouterRegistersHealthRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouter(applicationLogger)

	for _, path := range []string{"/api/v1/health"} {
		t.Run(path, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, path, nil)

			router.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
			}
		})
	}

	for _, path := range []string{"/heartbeat", "/probe", "/health"} {
		t.Run("removed "+path, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, path, nil)

			router.ServeHTTP(response, request)

			if response.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
			}
		})
	}
}

func TestHealthEndpointOverHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httptest.NewServer(newRouter(applicationLogger))
	t.Cleanup(server.Close)

	response, err := server.Client().Get(server.URL + "/api/v1/health")
	if err != nil {
		t.Fatalf("GET /api/v1/health: %v", err)
	}
	t.Cleanup(func() { response.Body.Close() })

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read health response: %v", err)
	}
	if strings.TrimSpace(string(body)) != `{"status":"ok"}` {
		t.Fatalf("health body = %q, want %q", body, `{"status":"ok"}`)
	}
}

func TestOpenAPIEndpointOverHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httptest.NewServer(newRouter(applicationLogger))
	t.Cleanup(server.Close)

	response, err := server.Client().Get(server.URL + "/api/v1/openapi.json")
	if err != nil {
		t.Fatalf("GET /api/v1/openapi.json: %v", err)
	}
	t.Cleanup(func() { response.Body.Close() })

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if contentType := response.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("content type = %q, want application/json", contentType)
	}

	var document struct {
		OpenAPI string                     `json:"openapi"`
		Paths   map[string]json.RawMessage `json:"paths"`
	}
	if err := json.NewDecoder(response.Body).Decode(&document); err != nil {
		t.Fatalf("decode OpenAPI response: %v", err)
	}
	if document.OpenAPI != "3.0.3" {
		t.Fatalf("openapi = %q, want %q", document.OpenAPI, "3.0.3")
	}
	for _, path := range []string{"/api/v1/health", "/api/v1/openapi.json"} {
		if _, ok := document.Paths[path]; !ok {
			t.Fatalf("OpenAPI response missing path %q", path)
		}
	}
}

func TestAPIV1ErrorsUseUnifiedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouter(applicationLogger)

	tests := []struct {
		name       string
		method     string
		path       string
		statusCode int
		code       api.ErrorCode
	}{
		{
			name:       "unknown route",
			method:     http.MethodGet,
			path:       "/api/v1/unknown",
			statusCode: http.StatusNotFound,
			code:       api.CodeNotFound,
		},
		{
			name:       "method not allowed",
			method:     http.MethodPost,
			path:       "/api/v1/health",
			statusCode: http.StatusMethodNotAllowed,
			code:       api.CodeMethodNotAllowed,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(test.method, test.path, nil)
			request.Header.Set("X-Request-ID", "contract-test-request")

			router.ServeHTTP(response, request)

			if response.Code != test.statusCode {
				t.Fatalf("status = %d, want %d", response.Code, test.statusCode)
			}
			if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
				t.Fatalf("content type = %q, want application/json", contentType)
			}
			if requestID := response.Header().Get("X-Request-ID"); requestID != "contract-test-request" {
				t.Fatalf("request ID = %q, want %q", requestID, "contract-test-request")
			}

			var body api.ErrorResponse
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if body.Code != test.code {
				t.Fatalf("error code = %q, want %q", body.Code, test.code)
			}
			if body.Message == "" {
				t.Fatal("error message must not be empty")
			}
		})
	}
}

func TestPaginationValidationMiddlewareIsRouteScoped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouter(applicationLogger)
	router.GET("/api/v1/contract-list", validatePaginationQuery(), func(context *gin.Context) {
		context.Status(http.StatusNoContent)
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/contract-list?page=0&page_size=101", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	var body api.ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Code != api.CodeInvalidPagination {
		t.Fatalf("error code = %q, want %q", body.Code, api.CodeInvalidPagination)
	}
}

func TestAPIV1PanicUsesSafeUnifiedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouter(applicationLogger)
	router.GET("/api/v1/panic", func(_ *gin.Context) {
		panic("sensitive panic detail")
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/panic", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	var body api.ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode recovery response: %v", err)
	}
	if body.Code != api.CodeInternal {
		t.Fatalf("error code = %q, want %q", body.Code, api.CodeInternal)
	}
	if strings.Contains(response.Body.String(), "sensitive panic detail") {
		t.Fatal("recovery response must not expose panic details")
	}
}

func TestNewHTTPServer(t *testing.T) {
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := newHTTPServer(applicationLogger, config.DefaultHTTPAddress)

	if server.Addr != config.DefaultHTTPAddress {
		t.Fatalf("server address = %q, want %q", server.Addr, config.DefaultHTTPAddress)
	}
	if server.Handler == nil {
		t.Fatal("server handler is nil")
	}
}
