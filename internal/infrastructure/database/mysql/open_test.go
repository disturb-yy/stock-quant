package mysql

import (
	"context"
	"strings"
	"testing"
)

func TestOpenRequiresDSN(t *testing.T) {
	db, err := Open(context.Background(), "")
	if err == nil || err.Error() != "DATABASE_DSN is required" {
		t.Fatalf("error = %v, want DATABASE_DSN is required", err)
	}
	if db != nil {
		t.Fatal("db = non-nil, want nil")
	}
}

func TestOpenRejectsMalformedDSNWithoutMySQL(t *testing.T) {
	db, err := Open(context.Background(), "malformed-dsn")
	if err == nil {
		t.Fatal("error = nil, want malformed DSN error")
	}
	if db != nil {
		t.Fatal("db = non-nil, want nil")
	}
	if !strings.Contains(err.Error(), "open mysql with gorm") && !strings.Contains(err.Error(), "ping mysql") {
		t.Fatalf("error = %v, want GORM open or ping context", err)
	}
}
