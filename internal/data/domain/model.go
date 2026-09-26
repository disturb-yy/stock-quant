package domain

import (
	"context"
	"fmt"
	"time"
)

const DateLayout = "2006-01-02"

const MaxPageSize = 50

type SyncTarget string

const (
	TargetBasicInfo SyncTarget = "basic_info"
	TargetDailyBars SyncTarget = "daily_bars"
	TargetAll       SyncTarget = "all"
)

func (target SyncTarget) Valid() bool {
	return target == TargetBasicInfo || target == TargetDailyBars || target == TargetAll
}

func (target SyncTarget) IncludesBasicInfo() bool {
	return target == TargetBasicInfo || target == TargetAll
}

func (target SyncTarget) IncludesDailyBars() bool {
	return target == TargetDailyBars || target == TargetAll
}

type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusRunning   TaskStatus = "running"
	StatusRetrying  TaskStatus = "retrying"
	StatusSucceeded TaskStatus = "succeeded"
	StatusFailed    TaskStatus = "failed"
)

type Trigger string

const (
	TriggerManual    Trigger = "manual"
	TriggerScheduled Trigger = "scheduled"
)

type SourceProvenance struct {
	Provider string
	Mode     string
}

func (source SourceProvenance) Valid() bool {
	return (source.Provider == "mock" && source.Mode == "mock") ||
		(source.Provider == "tushare" && source.Mode == "external")
}

type DateRange struct {
	Start *time.Time
	End   *time.Time
}

func NewDateRange(start, end *time.Time) (DateRange, error) {
	if (start == nil) != (end == nil) {
		return DateRange{}, ErrInvalidDateRange
	}
	if start == nil {
		return DateRange{}, nil
	}
	startDate := dateOnly(*start)
	endDate := dateOnly(*end)
	if endDate.Before(startDate) {
		return DateRange{}, ErrInvalidDateRange
	}
	return DateRange{Start: &startDate, End: &endDate}, nil
}

func ParseDateRange(start, end string) (DateRange, error) {
	if start == "" && end == "" {
		return DateRange{}, nil
	}
	parsedStart, err := time.Parse(DateLayout, start)
	if err != nil {
		return DateRange{}, fmt.Errorf("parse start date: %w", ErrInvalidDateRange)
	}
	parsedEnd, err := time.Parse(DateLayout, end)
	if err != nil {
		return DateRange{}, fmt.Errorf("parse end date: %w", ErrInvalidDateRange)
	}
	return NewDateRange(&parsedStart, &parsedEnd)
}

func (dateRange DateRange) ValidateFor(target SyncTarget) error {
	if !target.Valid() {
		return ErrInvalidTarget
	}
	if target.IncludesDailyBars() && (dateRange.Start == nil || dateRange.End == nil) {
		return ErrInvalidDateRange
	}
	if !target.IncludesDailyBars() && (dateRange.Start != nil || dateRange.End != nil) {
		return ErrInvalidDateRange
	}
	return nil
}

func (dateRange DateRange) Contains(date time.Time) bool {
	if dateRange.Start == nil || dateRange.End == nil {
		return false
	}
	day := dateOnly(date)
	return !day.Before(*dateRange.Start) && !day.After(*dateRange.End)
}

type SyncResult struct {
	ProcessedCount int
	CreatedCount   int
	UpdatedCount   int
	FailedCount    int
}

func (result SyncResult) HasPartialResult() bool {
	return result != (SyncResult{})
}

type SyncPayload struct {
	BasicInfo BasicInfoBatch
	DailyBars DailyBarBatch
	DataAsOf  *time.Time
}

type SyncTask struct {
	ID            string
	Target        SyncTarget
	Trigger       Trigger
	Status        TaskStatus
	Source        SourceProvenance
	DateRange     DateRange
	CreatedAt     time.Time
	StartedAt     *time.Time
	FinishedAt    *time.Time
	UpdatedAt     time.Time
	DataAsOf      *time.Time
	RetryCount    int
	MaxRetries    int
	FailureReason string
	Result        *SyncResult
	RetryOfTaskID *string
	NextAttemptAt *time.Time
}

type TaskPage struct {
	Items    []SyncTask
	Page     int
	PageSize int
	Total    int
}

func NewSyncTask(id string, target SyncTarget, dateRange DateRange, trigger Trigger,
	source SourceProvenance, maxRetries int, now time.Time, retryOf *string) (SyncTask, error) {
	task := SyncTask{
		ID: id, Target: target, Trigger: trigger, Status: StatusPending,
		Source: source, DateRange: dateRange, CreatedAt: now, UpdatedAt: now,
		MaxRetries: maxRetries, RetryOfTaskID: retryOf,
	}
	if err := task.Validate(); err != nil {
		return SyncTask{}, err
	}
	return task, nil
}

func (task SyncTask) Validate() error {
	if task.ID == "" || !task.Target.Valid() || !task.Source.Valid() || task.MaxRetries < 0 {
		return ErrInvalidTask
	}
	if task.Trigger != TriggerManual && task.Trigger != TriggerScheduled {
		return ErrInvalidTask
	}
	if err := task.DateRange.ValidateFor(task.Target); err != nil {
		return err
	}
	return nil
}

func (task *SyncTask) MarkRunning(now time.Time) error {
	if task.Status != StatusPending && task.Status != StatusRetrying {
		return ErrInvalidTask
	}
	task.Status = StatusRunning
	if task.StartedAt == nil {
		task.StartedAt = timePtr(now)
	}
	task.NextAttemptAt = nil
	task.UpdatedAt = now
	return nil
}

func (task *SyncTask) ScheduleRetry(now, nextAttempt time.Time, reason string) error {
	if task.Status != StatusRunning || task.RetryCount >= task.MaxRetries {
		return ErrInvalidTask
	}
	task.Status = StatusRetrying
	task.RetryCount++
	task.NextAttemptAt = timePtr(nextAttempt)
	task.FailureReason = reason
	task.UpdatedAt = now
	return nil
}

func (task *SyncTask) MarkFinished(status TaskStatus, now time.Time, result SyncResult,
	dataAsOf *time.Time, reason string) error {
	if task.Status != StatusRunning || (status != StatusSucceeded && status != StatusFailed) {
		return ErrInvalidTask
	}
	task.Status = status
	task.FinishedAt = timePtr(now)
	task.UpdatedAt = now
	task.DataAsOf = dataAsOf
	task.FailureReason = reason
	task.Result = &result
	// 失败任务只有已产生部分处理结果时才保存统计对象，避免把未处理伪装成空结果。
	if status == StatusFailed && !result.HasPartialResult() {
		task.Result = nil
	}
	task.NextAttemptAt = nil
	return nil
}

func (task SyncTask) CanManualRetry() bool {
	return task.Status == StatusFailed
}

type StockDataSource interface {
	Provenance() SourceProvenance
	FetchBasicInfo(context.Context) (BasicInfoBatch, error)
	FetchDailyBars(context.Context, DateRange) (DailyBarBatch, error)
}

type TaskRepository interface {
	CreateTask(context.Context, SyncTask) error
	FindTask(context.Context, string) (SyncTask, error)
	ListTasks(context.Context, int, int) (TaskPage, error)
	FindActiveTask(context.Context, SyncTarget) (SyncTask, error)
	ClaimNext(context.Context, time.Time) (SyncTask, error)
	CommitSuccess(context.Context, string, SyncPayload, time.Time) (SyncResult, error)
	CommitFailure(context.Context, string, SyncPayload, time.Time, string) (SyncResult, error)
	ScheduleRetry(context.Context, string, int, time.Time, time.Time, string) error
}

func dateOnly(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func timePtr(value time.Time) *time.Time {
	return &value
}
