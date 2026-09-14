package demo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeStatusReader struct {
	status DemoStatus
	err    error
}

func (reader fakeStatusReader) DemoStatus(context.Context) (DemoStatus, error) {
	return reader.status, reader.err
}

func TestRegisterRoutesReturnsDemoStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), fakeStatusReader{
		status: DemoStatus{Mode: "demo", Provider: "mysql-demo-fixture", SeedVersion: SeedVersion, Counts: Counts{Instruments: 3, DailyBars: 6, FinancialMetrics: 6}},
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/dev/demo-status", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("content type = %q, want application/json", contentType)
	}
	var body DemoStatus
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Mode != "demo" || body.SeedVersion != SeedVersion {
		t.Fatalf("body = %#v, want demo fixture status", body)
	}
}

func TestRegisterRoutesReturnsSafeDependencyError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), fakeStatusReader{err: errors.New("secret database detail")})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/dev/demo-status", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if strings.Contains(response.Body.String(), "secret database detail") {
		t.Fatal("dependency error must not be exposed")
	}
}
