package application

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/disturb-yy/stock-quant/internal/data/adapter/source"
	"github.com/disturb-yy/stock-quant/internal/data/domain"
)

type fakeRepository struct {
	tasks       map[string]domain.SyncTask
	active      map[domain.SyncTarget]string
	lastPayload domain.SyncPayload
	failSource  bool
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{tasks: make(map[string]domain.SyncTask), active: make(map[domain.SyncTarget]string)}
}

func (repository *fakeRepository) CreateTask(_ context.Context, task domain.SyncTask) error {
	if existing, ok := repository.active[task.Target]; ok {
		activeTask := repository.tasks[existing]
		if activeTask.Status == domain.StatusPending || activeTask.Status == domain.StatusRunning || activeTask.Status == domain.StatusRetrying {
			return domain.ErrTaskConflict
		}
	}
	repository.tasks[task.ID] = task
	repository.active[task.Target] = task.ID
	return nil
}

func (repository *fakeRepository) FindTask(_ context.Context, taskID string) (domain.SyncTask, error) {
	task, ok := repository.tasks[taskID]
	if !ok {
		return domain.SyncTask{}, domain.ErrTaskNotFound
	}
	return task, nil
}

func (repository *fakeRepository) ListTasks(_ context.Context, page, pageSize int) (domain.TaskPage, error) {
	if page < 1 || pageSize < 1 || pageSize > domain.MaxPageSize {
		return domain.TaskPage{}, domain.ErrInvalidPagination
	}
	items := make([]domain.SyncTask, 0, len(repository.tasks))
	for _, task := range repository.tasks {
		items = append(items, task)
	}
	start := (page - 1) * pageSize
	if start >= len(items) {
		items = []domain.SyncTask{}
	} else {
		end := start + pageSize
		if end > len(items) {
			end = len(items)
		}
		items = items[start:end]
	}
	return domain.TaskPage{Items: items, Page: page, PageSize: pageSize, Total: len(repository.tasks)}, nil
}

func (repository *fakeRepository) FindActiveTask(_ context.Context, target domain.SyncTarget) (domain.SyncTask, error) {
	taskID, ok := repository.active[target]
	if !ok {
		return domain.SyncTask{}, domain.ErrTaskNotFound
	}
	task, ok := repository.tasks[taskID]
	if !ok || (task.Status != domain.StatusPending && task.Status != domain.StatusRunning && task.Status != domain.StatusRetrying) {
		return domain.SyncTask{}, domain.ErrTaskNotFound
	}
	return task, nil
}

func (repository *fakeRepository) ClaimNext(_ context.Context, now time.Time) (domain.SyncTask, error) {
	for id, task := range repository.tasks {
		if (task.Status != domain.StatusPending && task.Status != domain.StatusRetrying) ||
			(task.NextAttemptAt != nil && task.NextAttemptAt.After(now)) {
			continue
		}
		if err := task.MarkRunning(now); err != nil {
			return domain.SyncTask{}, err
		}
		repository.tasks[id] = task
		return task, nil
	}
	return domain.SyncTask{}, domain.ErrNoDueTask
}

func (repository *fakeRepository) CommitSuccess(_ context.Context, taskID string,
	payload domain.SyncPayload, finishedAt time.Time) (domain.SyncResult, error) {
	task := repository.tasks[taskID]
	result := resultFor(payload)
	if err := task.MarkFinished(domain.StatusSucceeded, finishedAt, result, payload.DataAsOf, ""); err != nil {
		return domain.SyncResult{}, err
	}
	repository.tasks[taskID] = task
	repository.lastPayload = payload
	delete(repository.active, task.Target)
	return result, nil
}

func (repository *fakeRepository) CommitFailure(_ context.Context, taskID string,
	payload domain.SyncPayload, finishedAt time.Time, reason string) (domain.SyncResult, error) {
	task := repository.tasks[taskID]
	result := resultFor(payload)
	if err := task.MarkFinished(domain.StatusFailed, finishedAt, result, payload.DataAsOf, reason); err != nil {
		return domain.SyncResult{}, err
	}
	repository.tasks[taskID] = task
	repository.lastPayload = payload
	delete(repository.active, task.Target)
	return result, nil
}

func (repository *fakeRepository) ScheduleRetry(_ context.Context, taskID string, retryCount int,
	nextAttempt, updatedAt time.Time, reason string) error {
	task := repository.tasks[taskID]
	if task.Status != domain.StatusRunning {
		return domain.ErrInvalidTask
	}
	if err := task.ScheduleRetry(updatedAt, nextAttempt, reason); err != nil || task.RetryCount != retryCount {
		return errors.New("retry state mismatch")
	}
	repository.tasks[taskID] = task
	return nil
}

func resultFor(payload domain.SyncPayload) domain.SyncResult {
	processed := len(payload.BasicInfo.Items) + len(payload.DailyBars.Items)
	return domain.SyncResult{ProcessedCount: processed, CreatedCount: processed}
}

type fakeSource struct {
	delegate domain.StockDataSource
	fail     bool
}

func (source fakeSource) Provenance() domain.SourceProvenance { return source.delegate.Provenance() }

func (source fakeSource) FetchBasicInfo(ctx context.Context) (domain.BasicInfoBatch, error) {
	if source.fail {
		return domain.BasicInfoBatch{}, domain.ErrDataSourceUnavailable
	}
	return source.delegate.FetchBasicInfo(ctx)
}

func (source fakeSource) FetchDailyBars(ctx context.Context, dateRange domain.DateRange) (domain.DailyBarBatch, error) {
	if source.fail {
		return domain.DailyBarBatch{}, domain.ErrDataSourceUnavailable
	}
	return source.delegate.FetchDailyBars(ctx, dateRange)
}

type partialFailureSource struct {
	delegate domain.StockDataSource
}

func (source partialFailureSource) Provenance() domain.SourceProvenance {
	return source.delegate.Provenance()
}

func (source partialFailureSource) FetchBasicInfo(ctx context.Context) (domain.BasicInfoBatch, error) {
	return source.delegate.FetchBasicInfo(ctx)
}

func (source partialFailureSource) FetchDailyBars(context.Context, domain.DateRange) (domain.DailyBarBatch, error) {
	return domain.DailyBarBatch{}, domain.ErrDataSourceUnavailable
}

func TestServiceCreatesAndExecutesAllTarget(t *testing.T) {
	repository := newFakeRepository()
	clock := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	service, err := NewService(repository, source.NewMockAdapter(), 3, Options{
		Now: func() time.Time { return clock }, NewTaskID: func() string { return "task-1" },
	})
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	dateRange, err := domain.NewDateRange(&start, &end)
	if err != nil {
		t.Fatal(err)
	}
	task, err := service.CreateTask(context.Background(), CreateTaskInput{Target: domain.TargetAll, DateRange: dateRange})
	if err != nil {
		t.Fatal(err)
	}
	worked, err := service.ExecuteNext(context.Background())
	if err != nil || !worked {
		t.Fatalf("ExecuteNext = %v, %v", worked, err)
	}
	completed, err := repository.FindTask(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != domain.StatusSucceeded || completed.Result == nil || completed.Result.ProcessedCount != 6 {
		t.Fatalf("unexpected completed task: %#v", completed)
	}
	if completed.Source != source.NewMockAdapter().Provenance() {
		t.Fatalf("unexpected source: %#v", completed.Source)
	}
}

func TestServiceRetriesThenFailsAndAllowsManualRetry(t *testing.T) {
	repository := newFakeRepository()
	current := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	service, err := NewService(repository, fakeSource{delegate: source.NewMockAdapter(), fail: true}, 2, Options{
		Now: func() time.Time { return current }, NewTaskID: taskIDGenerator(), RetryBaseDelay: time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	task, err := service.CreateTask(context.Background(), CreateTaskInput{Target: domain.TargetBasicInfo})
	if err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 3; attempt++ {
		worked, err := service.ExecuteNext(context.Background())
		if err != nil || !worked {
			t.Fatalf("attempt %d ExecuteNext = %v, %v", attempt, worked, err)
		}
		current = current.Add(time.Second)
	}
	failed, err := repository.FindTask(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != domain.StatusFailed || failed.RetryCount != 2 || failed.FailureReason != "data source unavailable" {
		t.Fatalf("unexpected failed task: %#v", failed)
	}
	if failed.Result != nil {
		t.Fatalf("failed task without partial result should have nil Result: %#v", failed.Result)
	}
	retried, err := service.RetryFailedTask(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if retried.ID == task.ID || retried.RetryOfTaskID == nil || *retried.RetryOfTaskID != task.ID {
		t.Fatalf("unexpected manual retry: %#v", retried)
	}
}

func TestServiceRejectsDuplicateActiveTarget(t *testing.T) {
	repository := newFakeRepository()
	service, err := NewService(repository, source.NewMockAdapter(), 3, Options{
		NewTaskID: taskIDGenerator(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateTask(context.Background(), CreateTaskInput{Target: domain.TargetBasicInfo}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateTask(context.Background(), CreateTaskInput{Target: domain.TargetBasicInfo}); !errors.Is(err, domain.ErrTaskConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestServicePreservesPartialPayloadOnTerminalFailure(t *testing.T) {
	repository := newFakeRepository()
	service, err := NewService(repository, partialFailureSource{delegate: source.NewMockAdapter()}, 0, Options{
		NewTaskID: func() string { return "task-partial" },
	})
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	dateRange, err := domain.NewDateRange(&start, &end)
	if err != nil {
		t.Fatal(err)
	}
	task, err := service.CreateTask(context.Background(), CreateTaskInput{Target: domain.TargetAll, DateRange: dateRange})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ExecuteNext(context.Background()); err != nil {
		t.Fatal(err)
	}
	failed, err := repository.FindTask(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != domain.StatusFailed || failed.Result == nil || failed.Result.ProcessedCount != 2 {
		t.Fatalf("unexpected partial failure: %#v", failed)
	}
	if len(repository.lastPayload.BasicInfo.Items) != 2 || repository.lastPayload.DataAsOf == nil {
		t.Fatalf("partial payload was not preserved: %#v", repository.lastPayload)
	}
}

func taskIDGenerator() func() string {
	index := 0
	return func() string {
		index++
		return fmt.Sprintf("task-%d", index)
	}
}
