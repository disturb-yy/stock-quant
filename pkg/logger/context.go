package logger

import (
	"context"
	"log/slog"
)

type contextKey uint8

const (
	requestIDContextKey contextKey = iota
	requestLoggerContextKey
)

// WithRequestID stores a validated request ID in a request context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, requestID)
}

// RequestIDFromContext returns the request ID attached by a request middleware.
func RequestIDFromContext(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDContextKey).(string)
	return requestID, ok
}

// WithRequestLogger stores a request-scoped logger in a request context.
func WithRequestLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, requestLoggerContextKey, logger)
}

// RequestLoggerFromContext returns the request-scoped logger when one is available.
func RequestLoggerFromContext(ctx context.Context) (*slog.Logger, bool) {
	logger, ok := ctx.Value(requestLoggerContextKey).(*slog.Logger)
	return logger, ok && logger != nil
}
