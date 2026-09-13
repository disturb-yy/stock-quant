package logger

import (
	"context"
	"log/slog"
	"strings"
)

const redactedValue = "[REDACTED]"

var sensitiveFieldNames = []string{
	"password",
	"token",
	"authorization",
	"cookie",
	"secret",
	"api_key",
}

type redactingHandler struct {
	next slog.Handler
}

func newRedactingHandler(next slog.Handler) slog.Handler {
	return redactingHandler{next: next}
}

func (handler redactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return handler.next.Enabled(ctx, level)
}

func (handler redactingHandler) Handle(ctx context.Context, record slog.Record) error {
	redacted := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	record.Attrs(func(attr slog.Attr) bool {
		redacted.AddAttrs(redactAttribute(attr))
		return true
	})

	return handler.next.Handle(ctx, redacted)
}

func (handler redactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return redactingHandler{next: handler.next.WithAttrs(redactAttributes(attrs))}
}

func (handler redactingHandler) WithGroup(name string) slog.Handler {
	return redactingHandler{next: handler.next.WithGroup(name)}
}

func redactAttributes(attrs []slog.Attr) []slog.Attr {
	redacted := make([]slog.Attr, 0, len(attrs))
	for _, attr := range attrs {
		redacted = append(redacted, redactAttribute(attr))
	}

	return redacted
}

func redactAttribute(attr slog.Attr) slog.Attr {
	if containsSensitiveFieldName(attr.Key) {
		return slog.String(attr.Key, redactedValue)
	}

	value := attr.Value.Resolve()
	if value.Kind() != slog.KindGroup {
		return slog.Attr{Key: attr.Key, Value: value}
	}

	return slog.Attr{Key: attr.Key, Value: slog.GroupValue(redactAttributes(value.Group())...)}
}

func containsSensitiveFieldName(name string) bool {
	name = strings.ToLower(name)
	for _, sensitiveName := range sensitiveFieldNames {
		if strings.Contains(name, sensitiveName) {
			return true
		}
	}

	return false
}
