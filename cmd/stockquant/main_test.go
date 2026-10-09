package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"stock-quant/internal/app"
)

type checkerFunc func(context.Context) (app.HealthStatus, error)

func (f checkerFunc) Check(ctx context.Context) (app.HealthStatus, error) {
	return f(ctx)
}

type failedWriter struct{}

func (failedWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestRunHealthWritesProcessStatus(t *testing.T) {
	var output strings.Builder
	checkCalled := false
	check := checkerFunc(func(context.Context) (app.HealthStatus, error) {
		checkCalled = true
		return app.HealthStatus{Status: "ok", Scope: "process"}, nil
	})

	if err := run(context.Background(), []string{"health"}, &output, check); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if got, want := output.String(), "{\"status\":\"ok\",\"scope\":\"process\"}\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
	if !checkCalled {
		t.Fatal("health checker was not called")
	}
}

func TestRunHealthReturnsCheckerError(t *testing.T) {
	wantErr := errors.New("probe unavailable")
	check := checkerFunc(func(context.Context) (app.HealthStatus, error) {
		return app.HealthStatus{}, wantErr
	})

	err := run(context.Background(), []string{"health"}, &strings.Builder{}, check)
	if !errors.Is(err, wantErr) {
		t.Fatalf("run() error = %v, want wrapped %v", err, wantErr)
	}
}

func TestRunHealthReturnsOutputError(t *testing.T) {
	check := checkerFunc(func(context.Context) (app.HealthStatus, error) {
		return app.HealthStatus{Status: "ok", Scope: "process"}, nil
	})

	err := run(context.Background(), []string{"health"}, failedWriter{}, check)
	if err == nil || !strings.Contains(err.Error(), "write health output") {
		t.Fatalf("run() error = %v, want output error", err)
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	err := run(context.Background(), []string{"serve"}, &strings.Builder{}, nil)
	if err == nil || !strings.Contains(err.Error(), "usage: stockquant health") {
		t.Fatalf("run() error = %v, want usage error", err)
	}
}
