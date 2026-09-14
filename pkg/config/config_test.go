package config

import "testing"

func TestLoadServiceName(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		want        string
	}{
		{
			name: "default service name",
			want: "stock-quant",
		},
		{
			name:        "explicit service name",
			serviceName: "stock-quant-api",
			want:        "stock-quant-api",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("SERVICE_NAME", test.serviceName)

			if got := LoadServiceName(); got != test.want {
				t.Fatalf("LoadServiceName() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestLoadHTTPAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		want    string
		wantErr bool
	}{
		{
			name: "default address",
			want: DefaultHTTPAddress,
		},
		{
			name:    "explicit address",
			address: "127.0.0.1:18357",
			want:    "127.0.0.1:18357",
		},
		{
			name:    "missing port",
			address: "127.0.0.1",
			wantErr: true,
		},
		{
			name:    "invalid port",
			address: ":not-a-port",
			wantErr: true,
		},
		{
			name:    "out of range port",
			address: ":65536",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("HTTP_ADDRESS", test.address)

			got, err := LoadHTTPAddress()
			if test.wantErr {
				if err == nil {
					t.Fatal("LoadHTTPAddress() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadHTTPAddress() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("LoadHTTPAddress() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestLoadLogging(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		level       string
		format      string
		output      string
		directory   string
		want        Logging
	}{
		{
			name: "development defaults",
			want: Logging{
				Environment: "development",
				Level:       "info",
				Format:      "text",
				Output:      "console",
				Directory:   "logs",
			},
		},
		{
			name:        "production defaults to JSON",
			environment: "production",
			want: Logging{
				Environment: "production",
				Level:       "info",
				Format:      "json",
				Output:      "console",
				Directory:   "logs",
			},
		},
		{
			name:        "explicit values override defaults",
			environment: "staging",
			level:       "debug",
			format:      "json",
			output:      "file",
			directory:   "/var/log/stock-quant",
			want: Logging{
				Environment: "staging",
				Level:       "debug",
				Format:      "json",
				Output:      "file",
				Directory:   "/var/log/stock-quant",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("APP_ENV", test.environment)
			t.Setenv("LOG_LEVEL", test.level)
			t.Setenv("LOG_FORMAT", test.format)
			t.Setenv("LOG_OUTPUT", test.output)
			t.Setenv("LOG_DIR", test.directory)

			if got := LoadLogging(); got != test.want {
				t.Fatalf("LoadLogging() = %#v, want %#v", got, test.want)
			}
		})
	}
}
