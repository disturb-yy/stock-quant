package main

import (
	"strings"
	"testing"
)

func TestValidateMigrationCommandRestrictsDownEnvironment(t *testing.T) {
	tests := []struct {
		name      string
		action    string
		appEnv    string
		wantError string
	}{
		{name: "development down", action: "down", appEnv: "development"},
		{name: "test down", action: "down", appEnv: "test"},
		{name: "production down", action: "down", appEnv: "production", wantError: "only allowed"},
		{name: "staging down", action: "down", appEnv: "staging", wantError: "only allowed"},
		{name: "missing environment", action: "down", wantError: "APP_ENV"},
		{name: "unknown environment", action: "down", appEnv: "prod", wantError: "APP_ENV"},
		{name: "unknown action", action: "status", appEnv: "development", wantError: "usage"},
		{name: "production up", action: "up", appEnv: "production"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateMigrationCommand(test.action, test.appEnv)
			if test.wantError == "" && err != nil {
				t.Fatalf("validateMigrationCommand() error = %v", err)
			}
			if test.wantError != "" && (err == nil || !strings.Contains(err.Error(), test.wantError)) {
				t.Fatalf("validateMigrationCommand() error = %v, want substring %q", err, test.wantError)
			}
		})
	}
}
