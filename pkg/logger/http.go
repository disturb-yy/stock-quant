package logger

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	requestIDHeader    = "X-Request-ID"
	maxRequestIDLength = 128
)

// GinMiddleware adds request-scoped logging and emits one access log per response.
func GinMiddleware(logger *slog.Logger) gin.HandlerFunc {
	if logger == nil {
		panic("logger is required")
	}

	return func(context *gin.Context) {
		requestID, err := resolveRequestID(context.GetHeader(requestIDHeader))
		if err != nil {
			logger.Error("generate request ID", slog.Any("error", err))
			context.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		context.Header(requestIDHeader, requestID)
		requestLogger := logger.With(slog.String("request_id", requestID))
		requestContext := WithRequestID(context.Request.Context(), requestID)
		context.Request = context.Request.WithContext(WithRequestLogger(requestContext, requestLogger))

		startedAt := time.Now()
		context.Next()
		logGinResponse(requestLogger, context, context.Writer.Status(), time.Since(startedAt))
	}
}

func logGinResponse(logger *slog.Logger, context *gin.Context, status int, duration time.Duration) {
	attributes := []slog.Attr{
		slog.String("method", context.Request.Method),
		slog.String("path", context.Request.URL.Path),
		slog.Int("status", status),
		slog.Int64("duration_ms", duration.Milliseconds()),
	}
	if route := context.FullPath(); route != "" {
		attributes = append(attributes, slog.String("route", route))
	}

	switch {
	case status >= http.StatusInternalServerError:
		logger.LogAttrs(context.Request.Context(), slog.LevelError, "HTTP request completed", attributes...)
	case status >= http.StatusBadRequest:
		logger.LogAttrs(context.Request.Context(), slog.LevelWarn, "HTTP request completed", attributes...)
	default:
		logger.LogAttrs(context.Request.Context(), slog.LevelInfo, "HTTP request completed", attributes...)
	}
}

func resolveRequestID(candidate string) (string, error) {
	if isValidRequestID(candidate) {
		return candidate, nil
	}

	return newRequestID()
}

func isValidRequestID(requestID string) bool {
	if requestID == "" || len(requestID) > maxRequestIDLength {
		return false
	}

	for _, char := range requestID {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789._-", char) {
			return false
		}
	}

	return true
}

func newRequestID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("read random bytes for request ID: %w", err)
	}

	bytes[6] = bytes[6]&0x0f | 0x40
	bytes[8] = bytes[8]&0x3f | 0x80
	return formatUUID(bytes), nil
}

func formatUUID(bytes [16]byte) string {
	encoded := hex.EncodeToString(bytes[:])
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32]
}
