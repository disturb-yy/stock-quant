package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestNewWritesRootFieldsAndRedactsSecrets(t *testing.T) {
	var output bytes.Buffer
	logger, err := New(Config{
		Service:     "stock-quant",
		Environment: "production",
		Level:       "debug",
		Format:      "JSON",
		Output:      &output,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	logger.With(slog.String("session_token", "secret-token")).Debug(
		"request processed",
		slog.String("symbol", "600000.SH"),
		slog.Group("credentials", slog.String("api_key", "secret-key")),
	)

	entry := decodeJSONLog(t, output.Bytes())
	assertStringField(t, entry, "service", "stock-quant")
	assertStringField(t, entry, "environment", "production")
	assertStringField(t, entry, "level", "DEBUG")
	assertStringField(t, entry, "msg", "request processed")
	assertStringField(t, entry, "symbol", "600000.SH")
	assertStringField(t, entry, "session_token", redactedValue)

	credentials, ok := entry["credentials"].(map[string]any)
	if !ok {
		t.Fatalf("credentials = %#v, want JSON object", entry["credentials"])
	}
	assertStringField(t, credentials, "api_key", redactedValue)
}

func TestNewSupportsTextFormat(t *testing.T) {
	var output bytes.Buffer
	logger, err := New(Config{
		Service: "stock-quant",
		Format:  "TEXT",
		Output:  &output,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	logger.Info("text log")
	if !bytes.Contains(output.Bytes(), []byte(`msg="text log"`)) {
		t.Fatalf("text log output = %q, want text message", output.String())
	}
}

func TestNewRejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name:   "missing service",
			config: Config{Format: "text"},
		},
		{
			name:   "invalid log level",
			config: Config{Service: "stock-quant", Level: "verbose", Format: "text"},
		},
		{
			name:   "invalid log format",
			config: Config{Service: "stock-quant", Format: "xml"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := New(test.config); err == nil {
				t.Fatal("New() error = nil, want validation error")
			}
		})
	}
}

func decodeJSONLog(t *testing.T, raw []byte) map[string]any {
	t.Helper()

	var entry map[string]any
	if err := json.Unmarshal(raw, &entry); err != nil {
		t.Fatalf("decode JSON log %q: %v", raw, err)
	}

	return entry
}

func assertStringField(t *testing.T, entry map[string]any, key, want string) {
	t.Helper()

	if got, ok := entry[key].(string); !ok || got != want {
		t.Fatalf("log field %q = %#v, want %q", key, entry[key], want)
	}
}
