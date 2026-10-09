package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"stock-quant/contracts"
	"stock-quant/internal/market/domain"
	"stock-quant/internal/shared/apperror"
	"stock-quant/internal/shared/types"
)

type snapshotRepositoryFunc func(context.Context, types.TradingDate, int) (domain.Snapshot, error)

func (function snapshotRepositoryFunc) Load(ctx context.Context, asOf types.TradingDate, lookback int) (domain.Snapshot, error) {
	return function(ctx, asOf, lookback)
}

type factorRunnerFunc func(context.Context, contracts.FactorRequest) (contracts.FactorResult, error)

func (function factorRunnerFunc) Compute(ctx context.Context, request contracts.FactorRequest) (contracts.FactorResult, error) {
	return function(ctx, request)
}

func factorTask(t *testing.T) FactorTask {
	t.Helper()
	asOf, err := types.ParseTradingDate("2026-10-08")
	if err != nil {
		t.Fatal(err)
	}
	return FactorTask{
		RequestID:       "request-001",
		StrategyID:      "momentum_v1",
		StrategyVersion: "1.0.0",
		AsOf:            asOf,
		ConfigHash:      "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Params:          map[string]any{"top_n": 20},
	}
}

func factorSnapshot(t *testing.T) domain.Snapshot {
	t.Helper()
	asOf, err := types.ParseTradingDate("2026-10-08")
	if err != nil {
		t.Fatal(err)
	}
	return domain.Snapshot{
		AsOf:         asOf,
		SnapshotHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Reference: domain.SnapshotReference{
			Kind:   "local_file",
			Path:   "snapshots/2026-10-08.jsonl",
			SHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			Format: "jsonl",
		},
	}
}

func TestFactorServicePassesSnapshotAndRequestMetadataToRunner(t *testing.T) {
	task := factorTask(t)
	snapshot := factorSnapshot(t)
	want := contracts.FactorResult{
		SchemaVersion:   "v1",
		RequestID:       task.RequestID,
		StrategyID:      task.StrategyID,
		StrategyVersion: task.StrategyVersion,
		AsOf:            task.AsOf.String(),
		SnapshotHash:    snapshot.SnapshotHash,
		Status:          "success",
		Results:         []contracts.RawFactors{},
		Errors:          []contracts.ProtocolError{},
	}
	loaded := false
	runnerCalled := false
	service := NewFactorService(
		snapshotRepositoryFunc(func(_ context.Context, asOf types.TradingDate, lookback int) (domain.Snapshot, error) {
			loaded = true
			if asOf != task.AsOf || lookback != FactorLookbackDays {
				t.Fatalf("Load(asOf, lookback) = (%v, %d)", asOf, lookback)
			}
			return snapshot, nil
		}),
		factorRunnerFunc(func(_ context.Context, request contracts.FactorRequest) (contracts.FactorResult, error) {
			runnerCalled = true
			if request.RequestID != task.RequestID || request.SnapshotHash != snapshot.SnapshotHash {
				t.Fatalf("Compute() request identity = %#v", request)
			}
			if request.DataRef.Path != snapshot.Reference.Path || request.DataRef.SHA256 != snapshot.Reference.SHA256 {
				t.Fatalf("Compute() data reference = %#v", request.DataRef)
			}
			if request.Mode != "raw_factors" || request.AsOf != "2026-10-08" {
				t.Fatalf("Compute() protocol request = %#v", request)
			}
			return want, nil
		}),
	)

	got, err := service.ComputeRawFactors(context.Background(), task)
	if err != nil {
		t.Fatalf("ComputeRawFactors() error = %v", err)
	}
	if !loaded || !runnerCalled {
		t.Fatalf("dependencies called: snapshot=%t runner=%t", loaded, runnerCalled)
	}
	if got.RequestID != want.RequestID || got.SnapshotHash != want.SnapshotHash || got.Status != want.Status {
		t.Fatalf("ComputeRawFactors() = %#v, want %#v", got, want)
	}
}

func TestFactorServicePreservesSnapshotFailureCause(t *testing.T) {
	task := factorTask(t)
	cause := errors.New("snapshot file missing")
	service := NewFactorService(
		snapshotRepositoryFunc(func(context.Context, types.TradingDate, int) (domain.Snapshot, error) {
			return domain.Snapshot{}, cause
		}),
		factorRunnerFunc(func(context.Context, contracts.FactorRequest) (contracts.FactorResult, error) {
			t.Fatal("FactorRunner called after snapshot failure")
			return contracts.FactorResult{}, nil
		}),
	)

	_, err := service.ComputeRawFactors(context.Background(), task)
	if !errors.Is(err, cause) {
		t.Fatalf("ComputeRawFactors() error = %v, want wrapped snapshot error", err)
	}
	if got := apperror.CodeOf(err); got != apperror.CodeDataIncomplete {
		t.Fatalf("CodeOf() = %q, want %q", got, apperror.CodeDataIncomplete)
	}
}

func TestFactorServiceRejectsSnapshotFromAnotherDate(t *testing.T) {
	task := factorTask(t)
	wrongDate, err := types.ParseTradingDate("2026-10-07")
	if err != nil {
		t.Fatal(err)
	}
	snapshot := factorSnapshot(t)
	snapshot.AsOf = wrongDate
	runnerCalled := false
	service := NewFactorService(
		snapshotRepositoryFunc(func(context.Context, types.TradingDate, int) (domain.Snapshot, error) {
			return snapshot, nil
		}),
		factorRunnerFunc(func(context.Context, contracts.FactorRequest) (contracts.FactorResult, error) {
			runnerCalled = true
			return contracts.FactorResult{}, nil
		}),
	)

	_, err = service.ComputeRawFactors(context.Background(), task)
	if got := apperror.CodeOf(err); got != apperror.CodeDataIncomplete {
		t.Fatalf("CodeOf() = %q, want %q", got, apperror.CodeDataIncomplete)
	}
	if runnerCalled {
		t.Fatal("FactorRunner was called with a snapshot from another date")
	}
}

func TestFactorServicePreservesRunnerErrorCodeAndCause(t *testing.T) {
	task := factorTask(t)
	cause := errors.New("worker timed out")
	service := NewFactorService(
		snapshotRepositoryFunc(func(context.Context, types.TradingDate, int) (domain.Snapshot, error) {
			return factorSnapshot(t), nil
		}),
		factorRunnerFunc(func(context.Context, contracts.FactorRequest) (contracts.FactorResult, error) {
			return contracts.FactorResult{}, apperror.New(apperror.CodeTimeout, cause)
		}),
	)

	_, err := service.ComputeRawFactors(context.Background(), task)
	if !errors.Is(err, cause) {
		t.Fatalf("ComputeRawFactors() error = %v, want worker cause", err)
	}
	if got := apperror.CodeOf(err); got != apperror.CodeTimeout {
		t.Fatalf("CodeOf() = %q, want %q", got, apperror.CodeTimeout)
	}
}

func TestFactorServiceRejectsMismatchedRunnerResponse(t *testing.T) {
	task := factorTask(t)
	service := NewFactorService(
		snapshotRepositoryFunc(func(context.Context, types.TradingDate, int) (domain.Snapshot, error) {
			return factorSnapshot(t), nil
		}),
		factorRunnerFunc(func(context.Context, contracts.FactorRequest) (contracts.FactorResult, error) {
			return contracts.FactorResult{SchemaVersion: "v1", RequestID: "other-request"}, nil
		}),
	)

	_, err := service.ComputeRawFactors(context.Background(), task)
	if got := apperror.CodeOf(err); got != apperror.CodeInvalidWorkerResponse {
		t.Fatalf("CodeOf() = %q, want %q", got, apperror.CodeInvalidWorkerResponse)
	}
}

func TestFactorServicePropagatesContextCancellation(t *testing.T) {
	task := factorTask(t)
	ctx, cancel := context.WithCancel(context.Background())
	runnerEntered := make(chan struct{})
	service := NewFactorService(
		snapshotRepositoryFunc(func(context.Context, types.TradingDate, int) (domain.Snapshot, error) {
			return factorSnapshot(t), nil
		}),
		factorRunnerFunc(func(ctx context.Context, _ contracts.FactorRequest) (contracts.FactorResult, error) {
			close(runnerEntered)
			<-ctx.Done()
			return contracts.FactorResult{}, ctx.Err()
		}),
	)
	result := make(chan error, 1)
	go func() {
		_, err := service.ComputeRawFactors(ctx, task)
		result <- err
	}()
	<-runnerEntered
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ComputeRawFactors() error = %v, want context.Canceled", err)
		}
		if got := apperror.CodeOf(err); got != apperror.CodeCancelled {
			t.Fatalf("CodeOf() = %q, want %q", got, apperror.CodeCancelled)
		}
	case <-time.After(time.Second):
		t.Fatal("ComputeRawFactors() did not return after context cancellation")
	}
}
