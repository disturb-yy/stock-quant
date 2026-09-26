package config

import "testing"

func TestLoadDatabaseDSNFromEnv(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr string
	}{
		{
			name:  "configured",
			value: "test-dsn",
			want:  "test-dsn",
		},
		{
			name:    "missing",
			wantErr: "DATABASE_DSN is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DATABASE_DSN", tt.value)
			got, err := LoadDatabaseDSNFromEnv()
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("dsn = %q, want %q", got, tt.want)
			}
		})
	}
}
