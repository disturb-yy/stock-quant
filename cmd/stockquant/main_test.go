package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"stock-quant/internal/app"
	"stock-quant/internal/shared/types"
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

func TestRunInitialMarketSyncWritesSanitizedSummary(t *testing.T) {
	runner := &initialSyncRunnerFake{result: app.InitialMarketSyncResult{StockCount: 3, CalendarCount: 4}}
	var output strings.Builder
	err := runInitialMarketSyncCommand(context.Background(), []string{"--from-date", "2020-01-01", "--through-date", "2026-10-09"}, &output, runner)
	if err != nil {
		t.Fatalf("runInitialMarketSyncCommand() error = %v", err)
	}
	if got, want := output.String(), "initial market sync complete: stocks=3 calendars=4\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
	if runner.from.String() != "2020-01-01" || runner.through.String() != "2026-10-09" {
		t.Fatalf("sync date range = %s..%s", runner.from, runner.through)
	}
}

func TestRunInitialMarketSyncRejectsInvalidOrMissingRangeBeforeCallingRunner(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing through date", args: []string{"--from-date", "2020-01-01"}},
		{name: "invalid date", args: []string{"--from-date", "2020-02-30", "--through-date", "2020-03-01"}},
		{name: "reversed range", args: []string{"--from-date", "2021-01-01", "--through-date", "2020-01-01"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runner := &initialSyncRunnerFake{}
			err := runInitialMarketSyncCommand(context.Background(), test.args, &strings.Builder{}, runner)
			if err == nil || runner.called {
				t.Fatalf("command error=%v runner_called=%t; want rejected arguments before sync", err, runner.called)
			}
		})
	}
}

func TestRunInitialMarketSyncReturnsRunnerAndWriterErrors(t *testing.T) {
	args := []string{"--from-date", "2020-01-01", "--through-date", "2020-01-02"}
	wantErr := errors.New("database unavailable")
	runner := &initialSyncRunnerFake{err: wantErr}
	err := runInitialMarketSyncCommand(context.Background(), args, &strings.Builder{}, runner)
	if !errors.Is(err, wantErr) {
		t.Fatalf("runner error = %v, want %v", err, wantErr)
	}
	runner.err = nil
	if err := runInitialMarketSyncCommand(context.Background(), args, failedWriter{}, runner); err == nil || !strings.Contains(err.Error(), "write initial sync result") {
		t.Fatalf("writer error = %v, want summary write error", err)
	}
}

type initialSyncRunnerFake struct {
	result  app.InitialMarketSyncResult
	err     error
	from    types.TradingDate
	through types.TradingDate
	called  bool
}

func (f *initialSyncRunnerFake) Sync(_ context.Context, from, through types.TradingDate) (app.InitialMarketSyncResult, error) {
	f.called = true
	f.from = from
	f.through = through
	return f.result, f.err
}
