package mysql

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/disturb-yy/stock-quant/internal/data/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

var _ domain.StockQueryRepository = (*Repository)(nil)
var _ domain.TaskRepository = (*Repository)(nil)

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (repository *Repository) FindInstrument(ctx context.Context, identifier domain.StockIdentifier) (*domain.Instrument, error) {
	if err := repository.requireDatabase(); err != nil {
		return nil, err
	}
	var record instrumentRecord
	result := repository.db.WithContext(ctx).Where("market = ? AND symbol = ?", identifier.Market, identifier.Symbol).First(&record)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, queryUnavailable("find instrument", result.Error)
	}
	instrument, err := record.toDomain()
	if err != nil {
		return nil, queryUnavailable("map instrument", err)
	}
	return &instrument, nil
}

func (repository *Repository) FindDailyBars(ctx context.Context, identifier domain.StockIdentifier,
	dateRange domain.DateRange, limit int) ([]domain.DailyBar, error) {
	if err := repository.requireDatabase(); err != nil {
		return nil, err
	}
	if (dateRange.Start == nil) != (dateRange.End == nil) {
		return nil, domain.ErrInvalidDateRange
	}
	query := repository.db.WithContext(ctx).Model(&dailyBarRecord{}).
		Where("market = ? AND symbol = ?", identifier.Market, identifier.Symbol)
	if dateRange.Start != nil {
		query = query.Where("trade_date BETWEEN ? AND ?", dateOnly(dateRange.Start), dateOnly(dateRange.End)).
			Order("trade_date ASC")
	} else {
		if limit <= 0 {
			limit = domain.DefaultRecentTradingDays
		}
		recent := query.Select("*").Order("trade_date DESC").Limit(limit)
		query = repository.db.WithContext(ctx).Table("(?) AS recent_bars", recent).Order("trade_date ASC")
	}
	var records []dailyBarRecord
	if result := query.Find(&records); result.Error != nil {
		return nil, queryUnavailable("find daily bars", result.Error)
	}
	return mapDailyBars(records)
}

func (repository *Repository) CreateTask(ctx context.Context, task domain.SyncTask) error {
	if err := repository.requireDatabase(); err != nil {
		return err
	}
	if err := task.Validate(); err != nil {
		return err
	}
	return repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var activeTask taskRecord
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("active_target = ?", string(task.Target)).Take(&activeTask)
		if result.Error == nil {
			return domain.ErrTaskConflict
		}
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return writeUnavailable("check active sync task", result.Error)
		}
		if result := tx.Create(taskRecordFromDomain(task)); result.Error != nil {
			if isDuplicateKey(result.Error) {
				return domain.ErrTaskConflict
			}
			return writeUnavailable("insert sync task", result.Error)
		}
		return nil
	})
}

func (repository *Repository) FindTask(ctx context.Context, taskID string) (domain.SyncTask, error) {
	if err := repository.requireDatabase(); err != nil {
		return domain.SyncTask{}, err
	}
	var record taskRecord
	result := repository.db.WithContext(ctx).Where("id = ?", taskID).First(&record)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return domain.SyncTask{}, domain.ErrTaskNotFound
	}
	if result.Error != nil {
		return domain.SyncTask{}, queryUnavailable("find sync task", result.Error)
	}
	task, err := record.toDomain()
	if err != nil {
		return domain.SyncTask{}, mapTaskError("map sync task", err)
	}
	return task, nil
}

func (repository *Repository) ListTasks(ctx context.Context, page, pageSize int) (domain.TaskPage, error) {
	if err := repository.requireDatabase(); err != nil {
		return domain.TaskPage{}, err
	}
	offset, err := paginationOffset(page, pageSize)
	if err != nil {
		return domain.TaskPage{}, err
	}
	var total int64
	if result := repository.db.WithContext(ctx).Model(&taskRecord{}).Count(&total); result.Error != nil {
		return domain.TaskPage{}, queryUnavailable("count sync tasks", result.Error)
	}
	priority := clause.OrderBy{Expression: gorm.Expr("CASE WHEN status IN ? THEN 0 ELSE 1 END, updated_at DESC, id ASC",
		[]string{string(domain.StatusRunning), string(domain.StatusPending), string(domain.StatusRetrying)})}
	var records []taskRecord
	result := repository.db.WithContext(ctx).Model(&taskRecord{}).Order(priority).Limit(pageSize).Offset(offset).Find(&records)
	if result.Error != nil {
		return domain.TaskPage{}, queryUnavailable("list sync tasks", result.Error)
	}
	items, err := mapTasks(records)
	if err != nil {
		return domain.TaskPage{}, mapTaskError("map sync tasks", err)
	}
	return domain.TaskPage{Items: items, Page: page, PageSize: pageSize, Total: int(total)}, nil
}

func (repository *Repository) FindActiveTask(ctx context.Context, target domain.SyncTarget) (domain.SyncTask, error) {
	if err := repository.requireDatabase(); err != nil {
		return domain.SyncTask{}, err
	}
	var record taskRecord
	result := repository.db.WithContext(ctx).Where("active_target = ?", string(target)).First(&record)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return domain.SyncTask{}, domain.ErrTaskNotFound
	}
	if result.Error != nil {
		return domain.SyncTask{}, queryUnavailable("find active sync task", result.Error)
	}
	task, err := record.toDomain()
	if err != nil {
		return domain.SyncTask{}, mapTaskError("map active sync task", err)
	}
	return task, nil
}

func (repository *Repository) ClaimNext(ctx context.Context, now time.Time) (domain.SyncTask, error) {
	if err := repository.requireDatabase(); err != nil {
		return domain.SyncTask{}, err
	}
	var claimed domain.SyncTask
	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record taskRecord
		result := tx.Where("status IN ? AND (next_attempt_at IS NULL OR next_attempt_at <= ?)",
			[]string{string(domain.StatusPending), string(domain.StatusRetrying)}, now).
			Order("created_at ASC, id ASC").
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).First(&record)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return domain.ErrNoDueTask
		}
		if result.Error != nil {
			return queryUnavailable("select due sync task", result.Error)
		}
		var err error
		claimed, err = record.toDomain()
		if err != nil {
			return queryUnavailable("map due sync task", err)
		}
		if err := claimed.MarkRunning(now); err != nil {
			return err
		}
		updates := map[string]any{
			"status":          claimed.Status,
			"started_at":      claimed.StartedAt,
			"updated_at":      claimed.UpdatedAt,
			"next_attempt_at": nil,
		}
		if result := tx.Model(&taskRecord{}).Where("id = ?", claimed.ID).Updates(updates); result.Error != nil {
			return writeUnavailable("mark sync task running", result.Error)
		}
		return nil
	})
	if err != nil {
		return domain.SyncTask{}, err
	}
	return claimed, nil
}

func (repository *Repository) CommitSuccess(ctx context.Context, taskID string,
	payload domain.SyncPayload, finishedAt time.Time) (domain.SyncResult, error) {
	return repository.commitTerminal(ctx, taskID, payload, finishedAt, domain.StatusSucceeded, "")
}

func (repository *Repository) CommitFailure(ctx context.Context, taskID string,
	payload domain.SyncPayload, finishedAt time.Time, reason string) (domain.SyncResult, error) {
	return repository.commitTerminal(ctx, taskID, payload, finishedAt, domain.StatusFailed, reason)
}

func (repository *Repository) commitTerminal(ctx context.Context, taskID string, payload domain.SyncPayload,
	finishedAt time.Time, status domain.TaskStatus, reason string) (domain.SyncResult, error) {
	if err := repository.requireDatabase(); err != nil {
		return domain.SyncResult{}, err
	}
	var committed domain.SyncResult
	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record taskRecord
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", taskID).First(&record)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return domain.ErrTaskNotFound
		}
		if result.Error != nil {
			return queryUnavailable("lock sync task", result.Error)
		}
		if domain.TaskStatus(record.Status) != domain.StatusRunning {
			return domain.ErrInvalidTask
		}
		var err error
		committed, err = persistPayload(ctx, tx, payload)
		if err != nil {
			return err
		}
		dataAsOf := payload.DataAsOf
		if dataAsOf == nil {
			dataAsOf = payloadDataAsOf(payload)
		}
		updates := map[string]any{
			"status":          status,
			"active_target":   nil,
			"data_as_of":      dateOnly(dataAsOf),
			"finished_at":     finishedAt,
			"updated_at":      finishedAt,
			"failure_reason":  nullableReason(reason),
			"processed_count": committed.ProcessedCount,
			"created_count":   committed.CreatedCount,
			"updated_count":   committed.UpdatedCount,
			"failed_count":    committed.FailedCount,
		}
		if result := tx.Model(&taskRecord{}).Where("id = ?", taskID).Updates(updates); result.Error != nil {
			return writeUnavailable("finish sync task", result.Error)
		}
		return nil
	})
	if err != nil {
		return domain.SyncResult{}, err
	}
	return committed, nil
}

func (repository *Repository) ScheduleRetry(ctx context.Context, taskID string, retryCount int,
	nextAttempt, updatedAt time.Time, reason string) error {
	if err := repository.requireDatabase(); err != nil {
		return err
	}
	return repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record taskRecord
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", taskID).First(&record)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return domain.ErrTaskNotFound
		}
		if result.Error != nil {
			return queryUnavailable("lock retry task", result.Error)
		}
		if domain.TaskStatus(record.Status) != domain.StatusRunning {
			return domain.ErrInvalidTask
		}
		updates := map[string]any{
			"status":          domain.StatusRetrying,
			"retry_count":     retryCount,
			"next_attempt_at": nextAttempt,
			"updated_at":      updatedAt,
			"failure_reason":  nullableReason(reason),
		}
		if result := tx.Model(&taskRecord{}).Where("id = ?", taskID).Updates(updates); result.Error != nil {
			return writeUnavailable("schedule sync retry", result.Error)
		}
		return nil
	})
}

func persistPayload(ctx context.Context, tx *gorm.DB, payload domain.SyncPayload) (domain.SyncResult, error) {
	result, err := persistInstruments(ctx, tx, payload.BasicInfo.Items)
	if err != nil {
		return domain.SyncResult{}, err
	}
	barsResult, err := persistBars(ctx, tx, payload.DailyBars.Items)
	if err != nil {
		return domain.SyncResult{}, err
	}
	result.ProcessedCount += barsResult.ProcessedCount
	result.CreatedCount += barsResult.CreatedCount
	result.UpdatedCount += barsResult.UpdatedCount
	result.FailedCount += barsResult.FailedCount
	return result, nil
}

func persistInstruments(ctx context.Context, tx *gorm.DB, items []domain.Instrument) (domain.SyncResult, error) {
	result := domain.SyncResult{}
	for _, item := range items {
		result.ProcessedCount++
		record := instrumentRecordFromDomain(item)
		writeResult := tx.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "market"}, {Name: "symbol"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "status", "source_provider", "source_mode", "data_as_of", "updated_at"}),
		}).Create(&record)
		if writeResult.Error != nil {
			return domain.SyncResult{}, writeUnavailable("persist instrument", writeResult.Error)
		}
		result.CreatedCount += boolCount(writeResult.RowsAffected == 1)
		result.UpdatedCount += boolCount(writeResult.RowsAffected == 2)
	}
	return result, nil
}

func persistBars(ctx context.Context, tx *gorm.DB, items []domain.DailyBar) (domain.SyncResult, error) {
	result := domain.SyncResult{}
	for _, item := range items {
		result.ProcessedCount++
		record := dailyBarRecordFromDomain(item)
		writeResult := tx.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "market"}, {Name: "symbol"}, {Name: "trade_date"}},
			DoUpdates: clause.AssignmentColumns([]string{"open_price", "high_price", "low_price", "close_price", "volume", "source_provider", "source_mode", "updated_at"}),
		}).Create(&record)
		if writeResult.Error != nil {
			return domain.SyncResult{}, writeUnavailable("persist daily bar", writeResult.Error)
		}
		result.CreatedCount += boolCount(writeResult.RowsAffected == 1)
		result.UpdatedCount += boolCount(writeResult.RowsAffected == 2)
	}
	return result, nil
}

func (repository *Repository) requireDatabase() error {
	if repository == nil || repository.db == nil {
		return fmt.Errorf("mysql repository database is nil")
	}
	return nil
}

func paginationOffset(page, pageSize int) (int, error) {
	if page < 1 || pageSize < 1 || pageSize > domain.MaxPageSize {
		return 0, domain.ErrInvalidPagination
	}
	maxInt := int(^uint(0) >> 1)
	if page > 1 && page-1 > maxInt/pageSize {
		return 0, domain.ErrInvalidPagination
	}
	return (page - 1) * pageSize, nil
}

func payloadDataAsOf(payload domain.SyncPayload) *time.Time {
	if payload.BasicInfo.DataAsOf == nil {
		return payload.DailyBars.DataAsOf
	}
	if payload.DailyBars.DataAsOf == nil || !payload.DailyBars.DataAsOf.After(*payload.BasicInfo.DataAsOf) {
		return payload.BasicInfo.DataAsOf
	}
	return payload.DailyBars.DataAsOf
}

func dateOnly(value *time.Time) any {
	if value == nil {
		return nil
	}
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func nullableReason(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func boolCount(value bool) int {
	if value {
		return 1
	}
	return 0
}

func queryUnavailable(operation string, _ error) error {
	return fmt.Errorf("%s: %w", operation, domain.ErrDataSourceUnavailable)
}

func mapTaskError(operation string, err error) error {
	if errors.Is(err, domain.ErrInvalidDateRange) {
		return err
	}
	return queryUnavailable(operation, err)
}

func writeUnavailable(operation string, _ error) error {
	return fmt.Errorf("%s: %w", operation, domain.ErrDataSourceUnavailable)
}

func mapDailyBars(records []dailyBarRecord) ([]domain.DailyBar, error) {
	bars := make([]domain.DailyBar, 0, len(records))
	for _, record := range records {
		bar, err := record.toDomain()
		if err != nil {
			return nil, queryUnavailable("map daily bar", err)
		}
		bars = append(bars, bar)
	}
	return bars, nil
}

func mapTasks(records []taskRecord) ([]domain.SyncTask, error) {
	items := make([]domain.SyncTask, 0, len(records))
	for _, record := range records {
		task, err := record.toDomain()
		if err != nil {
			return nil, err
		}
		items = append(items, task)
	}
	return items, nil
}

func scanResult(status domain.TaskStatus, result domain.SyncResult) *domain.SyncResult {
	if status == domain.StatusSucceeded || (status == domain.StatusFailed && result.HasPartialResult()) {
		return &result
	}
	return nil
}

func isDuplicateKey(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "duplicate entry") || strings.Contains(message, "1062")
}
