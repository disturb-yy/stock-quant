package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNewRouterRegistersHealthRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouter(applicationLogger)

	for _, path := range []string{"/heartbeat", "/probe"} {
		t.Run(path, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, path, nil)

			router.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
			}
		})
	}
}

func TestNewHTTPServer(t *testing.T) {
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := newHTTPServer(applicationLogger)

	if server.Addr != defaultHTTPAddress {
		t.Fatalf("server address = %q, want %q", server.Addr, defaultHTTPAddress)
	}
	if server.Handler == nil {
		t.Fatal("server handler is nil")
	}
}
