package config

import "testing"

func TestLoadFromEnvDefaultsToMockAndThreeRetries(t *testing.T) {
	t.Setenv("DATA_SOURCE_PROVIDER", "")
	t.Setenv("TUSHARE_TOKEN", "")
	t.Setenv("TUSHARE_ENDPOINT", "")
	t.Setenv("MAX_RETRIES", "")
	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != ProviderMock || cfg.MaxRetries != 3 || cfg.TushareEndpoint != DefaultTushareURL {
		t.Fatalf("unexpected defaults: %#v", cfg)
	}
}

func TestLoadFromEnvRequiresTushareTokenWithoutEchoingIt(t *testing.T) {
	t.Setenv("DATA_SOURCE_PROVIDER", ProviderTushare)
	t.Setenv("TUSHARE_TOKEN", "")
	if _, err := LoadFromEnv(); err == nil {
		t.Fatal("expected missing token error")
	}
}

func TestLoadFromEnvParsesConfiguredRetries(t *testing.T) {
	t.Setenv("DATA_SOURCE_PROVIDER", ProviderMock)
	t.Setenv("MAX_RETRIES", "5")
	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxRetries != 5 {
		t.Fatalf("MaxRetries = %d, want 5", cfg.MaxRetries)
	}
}
