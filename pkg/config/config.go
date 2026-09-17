package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// DefaultHTTPAddress 是 HTTP 服务默认监听地址。
const DefaultHTTPAddress = ":8357"

// Database 包含连接本地 MySQL 所需的非业务配置。
type Database struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
}

// Tushare 包含 Tushare Pro 同步所需的运行时配置。
type Tushare struct {
	Token        string
	Endpoint     string
	StartDate    string
	EndDate      string
	LookbackDays int
	Timeout      time.Duration
}

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

// LoadEnvironment 读取应用运行环境；只有 development 会注册开发接口。
func LoadEnvironment() string {
	return strings.ToLower(valueOrDefault(os.Getenv("APP_ENV"), "development"))
}

// LoadDataProvider 读取 Provider 请求模式。默认使用本地演示 fixture。
func LoadDataProvider() string {
	return strings.ToLower(valueOrDefault(os.Getenv("DATA_PROVIDER"), "demo"))
}

// LoadTushare 读取 Tushare Pro 配置。Token 只从环境变量读取，不写入日志。
func LoadTushare() Tushare {
	return Tushare{
		Token:        strings.TrimSpace(os.Getenv("TUSHARE_TOKEN")),
		Endpoint:     valueOrDefault(os.Getenv("TUSHARE_ENDPOINT"), "https://api.tushare.pro"),
		StartDate:    strings.TrimSpace(os.Getenv("TUSHARE_START_DATE")),
		EndDate:      strings.TrimSpace(os.Getenv("TUSHARE_END_DATE")),
		LookbackDays: positiveIntOrDefault(os.Getenv("TUSHARE_LOOKBACK_DAYS"), 30),
		Timeout:      time.Duration(positiveIntOrDefault(os.Getenv("TUSHARE_TIMEOUT_SECONDS"), 15)) * time.Second,
	}
}

// LoadDatabase 读取本地 MySQL 连接配置。
func LoadDatabase() Database {
	return Database{
		Host:     valueOrDefault(os.Getenv("DB_HOST"), "127.0.0.1"),
		Port:     valueOrDefault(os.Getenv("DB_PORT"), "3307"),
		Name:     valueOrDefault(os.Getenv("DB_NAME"), "stock_quant_dev"),
		User:     valueOrDefault(os.Getenv("DB_USER"), "stock_quant"),
		Password: valueOrDefault(os.Getenv("DB_PASSWORD"), "stock_quant_dev"),
	}
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

func positiveIntOrDefault(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
