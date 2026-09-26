package domain

import (
	"errors"
	"testing"
	"time"
)

func TestDateRangeValidation(t *testing.T) {
	start := time.Date(2026, 9, 1, 8, 0, 0, 0, time.Local)
	end := time.Date(2026, 9, 2, 8, 0, 0, 0, time.Local)
	rangeValue, err := NewDateRange(&start, &end)
	if err != nil {
		t.Fatalf("NewDateRange returned error: %v", err)
	}
	if rangeValue.Start.Hour() != 0 || rangeValue.End.Hour() != 0 {
		t.Fatalf("date range was not normalized: %#v", rangeValue)
	}
	if err := rangeValue.ValidateFor(TargetDailyBars); err != nil {
		t.Fatalf("ValidateFor returned error: %v", err)
	}
	if err := (DateRange{}).ValidateFor(TargetDailyBars); !errors.Is(err, ErrInvalidDateRange) {
		t.Fatalf("expected invalid date range, got %v", err)
	}
}

func TestNewSyncTaskValidatesTargetAndDateRange(t *testing.T) {
	now := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	task, err := NewSyncTask("task-1", TargetBasicInfo, DateRange{}, TriggerManual,
		SourceProvenance{Provider: "mock", Mode: "mock"}, 3, now, nil)
	if err != nil {
		t.Fatalf("NewSyncTask returned error: %v", err)
	}
	if task.Status != StatusPending || task.RetryCount != 0 {
		t.Fatalf("unexpected initial task: %#v", task)
	}
	_, err = NewSyncTask("task-2", TargetDailyBars, DateRange{}, TriggerManual,
		SourceProvenance{Provider: "mock", Mode: "mock"}, 3, now, nil)
	if !errors.Is(err, ErrInvalidDateRange) {
		t.Fatalf("expected invalid date range, got %v", err)
	}
}

func TestSyncTaskLifecycle(t *testing.T) {
	now := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	task, err := NewSyncTask("task-1", TargetBasicInfo, DateRange{}, TriggerManual,
		SourceProvenance{Provider: "mock", Mode: "mock"}, 2, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := task.MarkRunning(now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := task.ScheduleRetry(now.Add(2*time.Minute), now.Add(4*time.Minute), "temporary"); err != nil {
		t.Fatal(err)
	}
	if task.Status != StatusRetrying || task.RetryCount != 1 {
		t.Fatalf("unexpected retry state: %#v", task)
	}
	if err := task.MarkRunning(now.Add(5 * time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := task.MarkFinished(StatusSucceeded, now.Add(6*time.Minute), SyncResult{}, nil, ""); err != nil {
		t.Fatal(err)
	}
	if task.Status != StatusSucceeded || task.Result == nil || task.FinishedAt == nil {
		t.Fatalf("unexpected final state: %#v", task)
	}
}

func TestSyncTaskFailedResultReflectsPartialProcessing(t *testing.T) {
	tests := []struct {
		name       string
		result     SyncResult
		wantResult bool
	}{
		{name: "no partial result", result: SyncResult{}, wantResult: false},
		{name: "partial result", result: SyncResult{ProcessedCount: 2, CreatedCount: 2}, wantResult: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			now := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
			task, err := NewSyncTask("task-1", TargetAll, DateRange{
				Start: timePtr(now.Add(-time.Hour)), End: timePtr(now),
			}, TriggerManual, SourceProvenance{Provider: "mock", Mode: "mock"}, 0, now, nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := task.MarkRunning(now); err != nil {
				t.Fatal(err)
			}
			if err := task.MarkFinished(StatusFailed, now.Add(time.Minute), test.result, nil, "failed"); err != nil {
				t.Fatal(err)
			}
			if (task.Result != nil) != test.wantResult {
				t.Fatalf("Result present = %t, want %t: %#v", task.Result != nil, test.wantResult, task.Result)
			}
		})
	}
}
