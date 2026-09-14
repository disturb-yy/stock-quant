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

// WithRequestID 将经过校验的请求 ID 写入请求上下文。
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, requestID)
}

// RequestIDFromContext 返回请求中间件附加的请求 ID。
func RequestIDFromContext(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDContextKey).(string)
	return requestID, ok
}

// WithRequestLogger 将请求级日志实例写入请求上下文。
func WithRequestLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, requestLoggerContextKey, logger)
}

// RequestLoggerFromContext 返回请求上下文中的请求级日志实例。
func RequestLoggerFromContext(ctx context.Context) (*slog.Logger, bool) {
	logger, ok := ctx.Value(requestLoggerContextKey).(*slog.Logger)
	return logger, ok && logger != nil
}
