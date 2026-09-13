package config

import (
	"os"
	"strings"
)

// Logging contains the runtime settings used to construct the application logger.
type Logging struct {
	Environment string
	Level       string
	Format      string
	Output      string
	Directory   string
}

// LoadServiceName reads the service name used by the application logger.
func LoadServiceName() string {
	return valueOrDefault(os.Getenv("SERVICE_NAME"), "stock-quant")
}

// LoadLogging reads logging settings from the process environment.
func LoadLogging() Logging {
	environment := environmentFromEnv()
	format := logFormatFromEnv(environment)

	return Logging{
		Environment: environment,
		Level:       valueOrDefault(os.Getenv("LOG_LEVEL"), "info"),
		Format:      format,
		Output:      valueOrDefault(os.Getenv("LOG_OUTPUT"), "console"),
		Directory:   valueOrDefault(os.Getenv("LOG_DIR"), "logs"),
	}
}

func environmentFromEnv() string {
	return valueOrDefault(os.Getenv("APP_ENV"), "development")
}

func logFormatFromEnv(environment string) string {
	if format := strings.TrimSpace(os.Getenv("LOG_FORMAT")); format != "" {
		return format
	}

	if strings.EqualFold(environment, "production") {
		return "json"
	}

	return "text"
}

func valueOrDefault(value, fallback string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}

	return fallback
}
