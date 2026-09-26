package main

import (
	"strings"
	"testing"
)

func TestRunRequiresDatabaseDSN(t *testing.T) {
	t.Setenv("DATABASE_DSN", "")
	if err := run(); err == nil || !strings.Contains(err.Error(), "DATABASE_DSN is required") {
		t.Fatalf("run error = %v, want missing DSN error", err)
	}
}
