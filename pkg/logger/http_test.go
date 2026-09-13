package logger

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/gin-gonic/gin"
)

var uuidV4Pattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestGinMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name            string
		clientRequestID string
		status          int
		wantLevel       string
		wantRequestID   string
	}{
		{
			name:          "generates ID and logs success",
			status:        http.StatusCreated,
			wantLevel:     "INFO",
			wantRequestID: "",
		},
		{
			name:            "propagates valid client ID and logs client error",
			clientRequestID: "client-request.1",
			status:          http.StatusNotFound,
			wantLevel:       "WARN",
			wantRequestID:   "client-request.1",
		},
		{
			name:            "replaces invalid client ID and logs server error",
			clientRequestID: "invalid request id",
			status:          http.StatusInternalServerError,
			wantLevel:       "ERROR",
			wantRequestID:   "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			logger, err := New(Config{
				Service:     "stock-quant",
				Environment: "test",
				Format:      "json",
				Output:      &output,
			})
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			var contextRequestID string
			var hasRequestLogger bool
			router := gin.New()
			router.Use(GinMiddleware(logger))
			router.GET("/stocks/:code", func(context *gin.Context) {
				contextRequestID, _ = RequestIDFromContext(context.Request.Context())
				_, hasRequestLogger = RequestLoggerFromContext(context.Request.Context())
				context.Status(test.status)
			})

			request := httptest.NewRequest(http.MethodGet, "/stocks/600000?access_token=leak", nil)
			request.Header.Set(requestIDHeader, test.clientRequestID)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != test.status {
				t.Fatalf("response status = %d, want %d", response.Code, test.status)
			}
			if !hasRequestLogger {
				t.Fatal("request-scoped logger was not added to context")
			}
			assertRequestID(t, contextRequestID, test.wantRequestID)
			if got := response.Header().Get(requestIDHeader); got != contextRequestID {
				t.Fatalf("response request ID = %q, want %q", got, contextRequestID)
			}

			entry := decodeJSONLog(t, output.Bytes())
			assertStringField(t, entry, "level", test.wantLevel)
			assertStringField(t, entry, "request_id", contextRequestID)
			assertStringField(t, entry, "method", http.MethodGet)
			assertStringField(t, entry, "path", "/stocks/600000")
			assertStringField(t, entry, "route", "/stocks/:code")
			if _, ok := entry["query"]; ok {
				t.Fatal("access log must not contain request query")
			}
			if got, ok := entry["status"].(float64); !ok || int(got) != test.status {
				t.Fatalf("status = %#v, want %d", entry["status"], test.status)
			}
			if _, ok := entry["duration_ms"].(float64); !ok {
				t.Fatalf("duration_ms = %#v, want numeric value", entry["duration_ms"])
			}
		})
	}
}

func TestGinMiddlewareDoesNotLogSensitiveRequestData(t *testing.T) {
	var output bytes.Buffer
	logger, err := New(Config{
		Service: "stock-quant",
		Format:  "json",
		Output:  &output,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	router := gin.New()
	router.Use(GinMiddleware(logger))
	router.POST("/login", func(context *gin.Context) {
		context.Status(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodPost, "/login?password=secret", nil)
	request.Header.Set("Authorization", "Bearer top-secret")
	request.Header.Set("Cookie", "session=top-secret")

	router.ServeHTTP(httptest.NewRecorder(), request)
	if got := string(output.Bytes()); containsSensitiveValue(got) {
		t.Fatalf("access log exposed request data: %q", got)
	}
}

func TestGinMiddlewareLogsAbortedStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var output bytes.Buffer
	applicationLogger, err := New(Config{
		Service: "stock-quant",
		Format:  "json",
		Output:  &output,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	router := gin.New()
	router.Use(GinMiddleware(applicationLogger))
	router.GET("/private", func(context *gin.Context) {
		context.AbortWithStatus(http.StatusUnauthorized)
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/private", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("response status = %d, want %d", response.Code, http.StatusUnauthorized)
	}

	entry := decodeJSONLog(t, output.Bytes())
	if got, ok := entry["status"].(float64); !ok || int(got) != http.StatusUnauthorized {
		t.Fatalf("status = %#v, want %d", entry["status"], http.StatusUnauthorized)
	}
}

func TestGinMiddlewareLogsRecoveredPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var output bytes.Buffer
	applicationLogger, err := New(Config{
		Service: "stock-quant",
		Format:  "json",
		Output:  &output,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	router := gin.New()
	router.Use(GinMiddleware(applicationLogger), gin.CustomRecovery(func(context *gin.Context, _ any) {
		context.AbortWithStatus(http.StatusInternalServerError)
	}))
	router.GET("/panic", func(_ *gin.Context) {
		panic("test panic")
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("response status = %d, want %d", response.Code, http.StatusInternalServerError)
	}

	entry := decodeJSONLog(t, output.Bytes())
	assertStringField(t, entry, "level", "ERROR")
	assertStringField(t, entry, "route", "/panic")
}

func assertRequestID(t *testing.T, got, want string) {
	t.Helper()

	if want != "" {
		if got != want {
			t.Fatalf("request ID = %q, want %q", got, want)
		}
		return
	}

	if !uuidV4Pattern.MatchString(got) {
		t.Fatalf("request ID = %q, want UUID v4", got)
	}
}

func containsSensitiveValue(value string) bool {
	return bytes.Contains([]byte(value), []byte("secret")) || bytes.Contains([]byte(value), []byte("top-secret"))
}
