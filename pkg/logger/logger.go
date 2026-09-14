package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Config 控制应用日志实例的输出方式和通用字段。
type Config struct {
	Service     string
	Environment string
	Level       string
	Format      string
	Output      io.Writer
}

// New 创建带通用服务字段和敏感信息脱敏能力的结构化日志实例。
func New(config Config) (*slog.Logger, error) {
	service := strings.TrimSpace(config.Service)
	if service == "" {
		return nil, fmt.Errorf("logger service is required")
	}

	level, err := parseLevel(config.Level)
	if err != nil {
		return nil, err
	}

	handler, err := newHandler(config.Format, config.Output, level)
	if err != nil {
		return nil, err
	}

	environment := valueOrDefault(config.Environment, "development")
	return slog.New(newRedactingHandler(handler).WithAttrs([]slog.Attr{
		slog.String("service", service),
		slog.String("environment", environment),
	})), nil
}

func parseLevel(value string) (slog.Level, error) {
	level := slog.LevelInfo
	value = valueOrDefault(value, "info")
	if err := level.UnmarshalText([]byte(strings.ToUpper(value))); err != nil {
		return 0, fmt.Errorf("parse log level %q: %w", value, err)
	}

	return level, nil
}

func newHandler(format string, output io.Writer, level slog.Level) (slog.Handler, error) {
	if output == nil {
		output = os.Stdout
	}

	options := &slog.HandlerOptions{Level: level}
	switch strings.ToLower(valueOrDefault(format, "text")) {
	case "text":
		return slog.NewTextHandler(output, options), nil
	case "json":
		return slog.NewJSONHandler(output, options), nil
	default:
		return nil, fmt.Errorf("unsupported log format %q", format)
	}
}

func valueOrDefault(value, fallback string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}

	return fallback
}
