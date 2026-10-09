package app

import (
	"context"
	"errors"
	"testing"
)

func TestHealthServiceRejectsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := (HealthService{}).Check(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Check() error = %v, want %v", err, context.Canceled)
	}
}

func TestHealthServiceReturnsProcessStatus(t *testing.T) {
	status, err := (HealthService{}).Check(context.Background())
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if status.Status != "ok" || status.Scope != "process" {
		t.Fatalf("Check() status = %#v, want process ok status", status)
	}
}
