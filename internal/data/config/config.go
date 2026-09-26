package config

import (
	"fmt"
	"os"
	"strconv"
)

const (
	DefaultMaxRetries = 3
	DefaultTushareURL = "https://api.tushare.pro"
	ProviderMock      = "mock"
	ProviderTushare   = "tushare"
)

type Config struct {
	Provider        string
	TushareToken    string
	TushareEndpoint string
	MaxRetries      int
}

func LoadFromEnv() (Config, error) {
	cfg := Config{
		Provider:        os.Getenv("DATA_SOURCE_PROVIDER"),
		TushareToken:    os.Getenv("TUSHARE_TOKEN"),
		TushareEndpoint: os.Getenv("TUSHARE_ENDPOINT"),
		MaxRetries:      DefaultMaxRetries,
	}
	if cfg.Provider == "" {
		cfg.Provider = ProviderMock
	}
	if cfg.TushareEndpoint == "" {
		cfg.TushareEndpoint = DefaultTushareURL
	}
	if raw := os.Getenv("MAX_RETRIES"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			return Config{}, fmt.Errorf("MAX_RETRIES must be a non-negative integer")
		}
		cfg.MaxRetries = value
	}
	if cfg.Provider != ProviderMock && cfg.Provider != ProviderTushare {
		return Config{}, fmt.Errorf("DATA_SOURCE_PROVIDER must be mock or tushare")
	}
	if cfg.Provider == ProviderTushare && cfg.TushareToken == "" {
		return Config{}, fmt.Errorf("TUSHARE_TOKEN is required when provider is tushare")
	}
	return cfg, nil
}
