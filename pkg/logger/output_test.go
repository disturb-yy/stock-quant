package logger

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenOutput(t *testing.T) {
	tests := []struct {
		name        string
		config      OutputConfig
		wantContent string
		wantError   bool
	}{
		{
			name:      "console output",
			config:    OutputConfig{Destination: "CONSOLE"},
			wantError: false,
		},
		{
			name: "file output",
			config: OutputConfig{
				Destination: "FILE",
				Directory:   "nested/logs",
				Service:     "stock-quant",
			},
			wantContent: "test log\n",
			wantError:   false,
		},
		{
			name:      "unsupported output",
			config:    OutputConfig{Destination: "syslog"},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := test.config
			if config.Directory == "nested/logs" {
				config.Directory = filepath.Join(t.TempDir(), config.Directory)
			}

			output, err := OpenOutput(config)
			if test.wantError {
				if err == nil {
					t.Fatal("OpenOutput() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("OpenOutput() error = %v", err)
			}

			if test.wantContent != "" {
				if _, err := io.WriteString(output, test.wantContent); err != nil {
					t.Fatalf("write log: %v", err)
				}
			}
			if err := output.Close(); err != nil {
				t.Fatalf("close output: %v", err)
			}

			if test.wantContent == "" {
				return
			}

			path := filepath.Join(config.Directory, "stock-quant.log")
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read log file: %v", err)
			}
			if !strings.Contains(string(content), test.wantContent) {
				t.Fatalf("log file content = %q, want %q", content, test.wantContent)
			}
		})
	}
}

func TestOpenOutputRejectsUnsafeServiceName(t *testing.T) {
	_, err := OpenOutput(OutputConfig{
		Destination: "file",
		Directory:   t.TempDir(),
		Service:     "../stock-quant",
	})
	if err == nil {
		t.Fatal("OpenOutput() error = nil, want error")
	}
}
