package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/disturb-yy/stock-quant/internal/data/domain"
)

type CreateTaskInput struct {
	Target    domain.SyncTarget
	DateRange domain.DateRange
	Trigger   domain.Trigger
}

type Options struct {
	Now            func() time.Time
	NewTaskID      func() string
	RetryBaseDelay time.Duration
}

type Service struct {
	repository     domain.TaskRepository
	source         domain.StockDataSource
	stockQuery     *StockQueryService
	maxRetries     int
	now            func() time.Time
	newTaskID      func() string
	retryBaseDelay time.Duration
}

func NewService(repository domain.TaskRepository, source domain.StockDataSource,
	maxRetries int, options Options) (*Service, error) {
	if repository == nil || source == nil || maxRetries < 0 || !source.Provenance().Valid() {
		return nil, fmt.Errorf("invalid data sync service configuration")
	}
	if options.Now == nil {
		options.Now = func() time.Time { return time.Now().UTC() }
	}
	if options.NewTaskID == nil {
		options.NewTaskID = newTaskID
	}
	if options.RetryBaseDelay <= 0 {
		options.RetryBaseDelay = time.Second
	}
	var stockQuery *StockQueryService
	if queryRepository, ok := repository.(domain.StockQueryRepository); ok {
		stockQuery, _ = NewStockQueryService(queryRepository)
	}
	return &Service{repository: repository, source: source, maxRetries: maxRetries,
		stockQuery: stockQuery, now: options.Now, newTaskID: options.NewTaskID,
		retryBaseDelay: options.RetryBaseDelay}, nil
}

func (service *Service) QueryStockData(ctx context.Context, input StockQueryInput) (domain.StockQueryResult, error) {
	if service.stockQuery == nil {
		return domain.StockQueryResult{}, domain.ErrDataSourceUnavailable
	}
	return service.stockQuery.QueryStockData(ctx, input)
}

func (service *Service) CreateTask(ctx context.Context, input CreateTaskInput) (domain.SyncTask, error) {
	if input.Trigger == "" {
		input.Trigger = domain.TriggerManual
	}
	now := service.now()
	task, err := domain.NewSyncTask(service.newTaskID(), input.Target, input.DateRange, input.Trigger,
		service.source.Provenance(), service.maxRetries, now, nil)
	if err != nil {
		return domain.SyncTask{}, err
	}
	if err := service.repository.CreateTask(ctx, task); err != nil {
		return domain.SyncTask{}, err
	}
	return task, nil
}

func (service *Service) RetryFailedTask(ctx context.Context, taskID string) (domain.SyncTask, error) {
	original, err := service.repository.FindTask(ctx, taskID)
	if err != nil {
		return domain.SyncTask{}, err
	}
	if !original.CanManualRetry() {
		return domain.SyncTask{}, domain.ErrTaskNotRetryable
	}
	now := service.now()
	retryTask, err := domain.NewSyncTask(service.newTaskID(), original.Target, original.DateRange,
		domain.TriggerManual, service.source.Provenance(), service.maxRetries, now, &original.ID)
	if err != nil {
		return domain.SyncTask{}, err
	}
	if err := service.repository.CreateTask(ctx, retryTask); err != nil {
		return domain.SyncTask{}, err
	}
	return retryTask, nil
}

func (service *Service) GetTask(ctx context.Context, taskID string) (domain.SyncTask, error) {
	return service.repository.FindTask(ctx, taskID)
}

func (service *Service) ListTasks(ctx context.Context, page, pageSize int) (domain.TaskPage, error) {
	if page < 1 || pageSize < 1 || pageSize > domain.MaxPageSize {
		return domain.TaskPage{}, domain.ErrInvalidPagination
	}
	return service.repository.ListTasks(ctx, page, pageSize)
}

func (service *Service) ExecuteNext(ctx context.Context) (bool, error) {
	task, err := service.repository.ClaimNext(ctx, service.now())
	if errors.Is(err, domain.ErrNoDueTask) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := service.executeTask(ctx, task); err != nil {
		return true, err
	}
	return true, nil
}

func (service *Service) executeTask(ctx context.Context, task domain.SyncTask) error {
	payload := domain.SyncPayload{}
	if task.Target.IncludesBasicInfo() {
		batch, err := service.source.FetchBasicInfo(ctx)
		if err != nil {
			return service.handleFailure(ctx, task, payload, err)
		}
		payload.BasicInfo = batch
	}
	if task.Target.IncludesDailyBars() {
		batch, err := service.source.FetchDailyBars(ctx, task.DateRange)
		if err != nil {
			return service.handleFailure(ctx, task, payload, err)
		}
		payload.DailyBars = batch
	}
	if err := validatePayload(task, payload, service.source.Provenance()); err != nil {
		return service.handleFailure(ctx, task, payload, err)
	}
	payload.DataAsOf = latestPayloadDate(payload)
	if _, err := service.repository.CommitSuccess(ctx, task.ID, payload, service.now()); err != nil {
		return service.handleFailure(ctx, task, domain.SyncPayload{}, err)
	}
	return nil
}

func (service *Service) handleFailure(ctx context.Context, task domain.SyncTask,
	payload domain.SyncPayload, cause error) error {
	now := service.now()
	reason := failureReason(cause)
	if payload.DataAsOf == nil {
		payload.DataAsOf = latestPayloadDate(payload)
	}
	if task.RetryCount < task.MaxRetries {
		next := now.Add(service.retryBaseDelay * time.Duration(1<<task.RetryCount))
		return service.repository.ScheduleRetry(ctx, task.ID, task.RetryCount+1, next, now, reason)
	}
	_, err := service.repository.CommitFailure(ctx, task.ID, payload, now, reason)
	return err
}

func validatePayload(task domain.SyncTask, payload domain.SyncPayload,
	source domain.SourceProvenance) error {
	if task.Target.IncludesBasicInfo() {
		if err := payload.BasicInfo.Validate(); err != nil {
			return err
		}
		if !batchSourceMatches(payload.BasicInfo.Items, source) {
			return fmt.Errorf("basic info source mismatch")
		}
	}
	if task.Target.IncludesDailyBars() {
		if err := payload.DailyBars.Validate(task.DateRange); err != nil {
			return err
		}
		if !barSourceMatches(payload.DailyBars.Items, source) {
			return fmt.Errorf("daily bars source mismatch")
		}
	}
	return nil
}

func batchSourceMatches(items []domain.Instrument, expected domain.SourceProvenance) bool {
	for _, item := range items {
		if item.Source != expected {
			return false
		}
	}
	return true
}

func barSourceMatches(items []domain.DailyBar, expected domain.SourceProvenance) bool {
	for _, item := range items {
		if item.Source != expected {
			return false
		}
	}
	return true
}

func latestPayloadDate(payload domain.SyncPayload) *time.Time {
	var latest *time.Time
	if payload.BasicInfo.DataAsOf != nil {
		value := *payload.BasicInfo.DataAsOf
		latest = &value
	}
	if payload.DailyBars.DataAsOf != nil && (latest == nil || payload.DailyBars.DataAsOf.After(*latest)) {
		value := *payload.DailyBars.DataAsOf
		latest = &value
	}
	return latest
}

func failureReason(err error) string {
	if errors.Is(err, domain.ErrDataSourceUnavailable) {
		return "data source unavailable"
	}
	if errors.Is(err, context.Canceled) {
		return "execution canceled"
	}
	return "sync execution failed"
}

func newTaskID() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return fmt.Sprintf("task-%d", time.Now().UnixNano())
	}
	encoded := hex.EncodeToString(value)
	return encoded[:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:]
}
