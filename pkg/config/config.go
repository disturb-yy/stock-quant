package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

// DefaultHTTPAddress 是 HTTP 服务默认监听地址。
const DefaultHTTPAddress = ":8357"

// Logging 包含构造应用日志实例所需的运行时配置。
type Logging struct {
	Environment string
	Level       string
	Format      string
	Output      string
	Directory   string
}

// LoadServiceName 读取应用日志使用的服务名称。
func LoadServiceName() string {
	return valueOrDefault(os.Getenv("SERVICE_NAME"), "stock-quant")
}

// LoadHTTPAddress 读取并校验 HTTP 服务的监听地址。
func LoadHTTPAddress() (string, error) {
	address := valueOrDefault(os.Getenv("HTTP_ADDRESS"), DefaultHTTPAddress)
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", fmt.Errorf("parse HTTP_ADDRESS %q: %w", address, err)
	}

	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return "", fmt.Errorf("parse HTTP_ADDRESS %q: port must be between 1 and 65535", address)
	}

	return address, nil
}

// LoadLogging 从进程环境变量中读取日志配置。
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
