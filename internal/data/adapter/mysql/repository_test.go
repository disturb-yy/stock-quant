package mysql

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/disturb-yy/stock-quant/internal/data/domain"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func newMockRepository(t *testing.T) (*Repository, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(gormmysql.New(gormmysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		_ = sqlDB.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
		_ = sqlDB.Close()
	})
	return NewRepository(db), mock
}

func TestFindInstrumentUsesGORMAndMapsNullableDate(t *testing.T) {
	repository, mock := newMockRepository(t)
	updatedAt := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	dataAsOf := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT .*t_instrument.*").
		WithArgs("SH", "600519", 1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "symbol", "market", "name", "status", "source_provider", "source_mode", "data_as_of", "updated_at",
		}).AddRow(1, "600519", "SH", "贵州茅台", "active", "mock", "mock", dataAsOf, updatedAt))

	got, err := repository.FindInstrument(context.Background(), domain.StockIdentifier{Symbol: "600519", Market: "SH"})
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Name != "贵州茅台" || got.DataAsOf == nil || !got.DataAsOf.Equal(dataAsOf) {
		t.Fatalf("unexpected instrument: %#v", got)
	}
}

func TestFindInstrumentMapsDatabaseErrorToUnavailable(t *testing.T) {
	repository, mock := newMockRepository(t)
	mock.ExpectQuery("SELECT .*t_instrument.*").
		WithArgs("SH", "600519", 1).
		WillReturnError(errors.New("SELECT * FROM t_instrument failed"))

	_, err := repository.FindInstrument(context.Background(), domain.StockIdentifier{Symbol: "600519", Market: "SH"})
	if !errors.Is(err, domain.ErrDataSourceUnavailable) {
		t.Fatalf("FindInstrument error = %v, want data source unavailable", err)
	}
	if strings.Contains(err.Error(), "t_instrument") || strings.Contains(err.Error(), "SELECT") {
		t.Fatalf("FindInstrument error leaks SQL details: %v", err)
	}
}

func TestFindDailyBarsUsesGORMForExplicitRange(t *testing.T) {
	repository, mock := newMockRepository(t)
	tradeDate := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT .*t_instrument_daily_bar.*").
		WithArgs("SZ", "000001", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "symbol", "market", "trade_date", "open_price", "high_price", "low_price", "close_price", "volume",
			"source_provider", "source_mode", "updated_at",
		}).AddRow(1, "000001", "SZ", tradeDate, 10.0, 10.5, 9.8, 10.2, 100000.0, "mock", "mock", updatedAt))

	start := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	dateRange, err := domain.NewDateRange(&start, &end)
	if err != nil {
		t.Fatal(err)
	}
	bars, err := repository.FindDailyBars(context.Background(), domain.StockIdentifier{Symbol: "000001", Market: "SZ"}, dateRange, 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(bars) != 1 || !bars[0].TradeDate.Equal(tradeDate) || bars[0].Close != 10.2 {
		t.Fatalf("unexpected daily bars: %#v", bars)
	}
}

func TestFindDailyBarsUsesRecentBarsSubqueryByDefault(t *testing.T) {
	repository, mock := newMockRepository(t)
	firstDate := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	secondDate := time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT .*t_instrument_daily_bar.*recent_bars.*").
		WithArgs("SH", "600519", domain.DefaultRecentTradingDays).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "symbol", "market", "trade_date", "open_price", "high_price", "low_price", "close_price", "volume",
			"source_provider", "source_mode", "updated_at",
		}).AddRow(1, "600519", "SH", firstDate, 10.0, 10.5, 9.8, 10.2, 100000.0, "mock", "mock", updatedAt).
			AddRow(2, "600519", "SH", secondDate, 10.2, 10.8, 10.0, 10.6, 120000.0, "mock", "mock", updatedAt))

	bars, err := repository.FindDailyBars(context.Background(), domain.StockIdentifier{Symbol: "600519", Market: "SH"}, domain.DateRange{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(bars) != 2 || !bars[0].TradeDate.Equal(firstDate) || !bars[1].TradeDate.Equal(secondDate) {
		t.Fatalf("unexpected recent bars: %#v", bars)
	}
}

func TestFindDailyBarsRejectsPartialDateRange(t *testing.T) {
	repository, _ := newMockRepository(t)
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	_, err := repository.FindDailyBars(context.Background(), domain.StockIdentifier{Symbol: "600519", Market: "SH"}, domain.DateRange{Start: &start}, 30)
	if !errors.Is(err, domain.ErrInvalidDateRange) {
		t.Fatalf("FindDailyBars error = %v, want invalid date range", err)
	}
}

func TestCreateTaskUsesTransactionAndConflictCheck(t *testing.T) {
	repository, mock := newMockRepository(t)
	now := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	task, err := domain.NewSyncTask("task-1", domain.TargetBasicInfo, domain.DateRange{}, domain.TriggerManual,
		domain.SourceProvenance{Provider: "mock", Mode: "mock"}, 2, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*t_data_sync_task.*active_target.*FOR UPDATE").
		WithArgs("basic_info", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectExec("INSERT INTO .*t_data_sync_task.*").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := repository.CreateTask(context.Background(), task); err != nil {
		t.Fatal(err)
	}
}

func TestClaimNextUsesSkipLockedAndUpdatesWithinTransaction(t *testing.T) {
	repository, mock := newMockRepository(t)
	now := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*t_data_sync_task.*FOR UPDATE SKIP LOCKED").
		WithArgs("pending", "retrying", now, 1).
		WillReturnRows(taskRows(now).AddRow(
			"task-1", "basic_info", "manual", "pending", "basic_info", "mock", "mock", nil, nil, nil,
			now, nil, nil, now, nil, 0, 2, nil, 0, 0, 0, 0, nil,
		))
	mock.ExpectExec("UPDATE .*t_data_sync_task.*").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	task, err := repository.ClaimNext(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != domain.StatusRunning || task.StartedAt == nil || task.NextAttemptAt != nil {
		t.Fatalf("unexpected claimed task: %#v", task)
	}
}

func TestClaimNextReturnsNoDueTask(t *testing.T) {
	repository, mock := newMockRepository(t)
	now := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*t_data_sync_task.*FOR UPDATE SKIP LOCKED").
		WithArgs("pending", "retrying", now, 1).
		WillReturnError(gorm.ErrRecordNotFound)
	mock.ExpectRollback()

	_, err := repository.ClaimNext(context.Background(), now)
	if !errors.Is(err, domain.ErrNoDueTask) {
		t.Fatalf("ClaimNext error = %v, want no due task", err)
	}
}

func TestCommitSuccessPersistsWithGORMUpsertAndCountsRows(t *testing.T) {
	repository, mock := newMockRepository(t)
	now := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*t_data_sync_task.*FOR UPDATE").
		WithArgs("task-1", 1).
		WillReturnRows(taskRows(now).AddRow(
			"task-1", "all", "manual", "running", "all", "mock", "mock", time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC), nil, now, now, nil, now, nil, 0, 2, nil, 0, 0, 0, 0, nil,
		))
	mock.ExpectExec("INSERT INTO .*t_instrument.*ON DUPLICATE KEY UPDATE.*").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO .*t_instrument_daily_bar.*ON DUPLICATE KEY UPDATE.*").WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec("UPDATE .*t_data_sync_task.*").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	dataAsOf := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	result, err := repository.CommitSuccess(context.Background(), "task-1", domain.SyncPayload{
		BasicInfo: domain.BasicInfoBatch{Items: []domain.Instrument{{
			Symbol: "600519", Market: "SH", Name: "贵州茅台", Status: "active",
			Source: domain.SourceProvenance{Provider: "mock", Mode: "mock"}, UpdatedAt: now,
		}}, DataAsOf: &dataAsOf},
		DailyBars: domain.DailyBarBatch{Items: []domain.DailyBar{{
			Symbol: "600519", Market: "SH", TradeDate: dataAsOf, Open: 10, High: 11, Low: 9, Close: 10.5,
			Volume: 100, Source: domain.SourceProvenance{Provider: "mock", Mode: "mock"}, UpdatedAt: now,
		}}},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if result.ProcessedCount != 2 || result.CreatedCount != 1 || result.UpdatedCount != 1 {
		t.Fatalf("unexpected sync result: %#v", result)
	}
}

func TestCommitTerminalRejectsNonRunningTask(t *testing.T) {
	repository, mock := newMockRepository(t)
	now := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*t_data_sync_task.*FOR UPDATE").
		WithArgs("task-1", 1).
		WillReturnRows(taskRows(now).AddRow(
			"task-1", "basic_info", "manual", "pending", "basic_info", "mock", "mock", nil, nil, nil,
			now, nil, nil, now, nil, 0, 2, nil, 0, 0, 0, 0, nil,
		))
	mock.ExpectRollback()

	_, err := repository.CommitSuccess(context.Background(), "task-1", domain.SyncPayload{}, now)
	if !errors.Is(err, domain.ErrInvalidTask) {
		t.Fatalf("CommitSuccess error = %v, want invalid task", err)
	}
}

func TestScheduleRetryLocksAndUpdatesTask(t *testing.T) {
	repository, mock := newMockRepository(t)
	now := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	next := now.Add(time.Minute)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*t_data_sync_task.*FOR UPDATE").
		WithArgs("task-1", 1).
		WillReturnRows(taskRows(now).AddRow(
			"task-1", "basic_info", "manual", "running", "basic_info", "mock", "mock", nil, nil, nil,
			now, now, nil, now, nil, 0, 2, nil, 0, 0, 0, 0, nil,
		))
	mock.ExpectExec("UPDATE .*t_data_sync_task.*").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := repository.ScheduleRetry(context.Background(), "task-1", 1, next, now, "temporary"); err != nil {
		t.Fatal(err)
	}
}

func TestListTasksUsesGORMCountAndPriorityOrdering(t *testing.T) {
	repository, mock := newMockRepository(t)
	now := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM .*t_data_sync_task.*").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT .*t_data_sync_task.*ORDER BY CASE WHEN status IN.*").
		WithArgs("running", "pending", "retrying", 10).
		WillReturnRows(taskRows(now).AddRow(
			"task-1", "basic_info", "manual", "pending", "basic_info", "mock", "mock", nil, nil, nil,
			now, nil, nil, now, nil, 0, 2, nil, 0, 0, 0, 0, nil,
		))

	page, err := repository.ListTasks(context.Background(), 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].ID != "task-1" {
		t.Fatalf("unexpected task page: %#v", page)
	}
}

func TestTaskRecordMapsNullableFieldsAndPartialResult(t *testing.T) {
	now := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	reason := "partial failure"
	retryOf := "task-0"
	task, err := (taskRecord{
		ID: "task-1", Target: "all", TriggerType: "manual", Status: "failed", SourceProvider: "mock", SourceMode: "mock",
		StartDate: &start, EndDate: &end, CreatedAt: now, UpdatedAt: now, FailureReason: &reason,
		ProcessedCount: 2, CreatedCount: 1, UpdatedCount: 1, RetryOfTaskID: &retryOf,
	}).toDomain()
	if err != nil {
		t.Fatal(err)
	}
	if task.DateRange.Start == nil || task.RetryOfTaskID == nil || task.Result == nil || task.Result.ProcessedCount != 2 {
		t.Fatalf("unexpected task mapping: %#v", task)
	}
}

func TestTaskRecordRejectsPartialDateRange(t *testing.T) {
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	_, err := (taskRecord{StartDate: &start}).toDomain()
	if !errors.Is(err, domain.ErrInvalidDateRange) {
		t.Fatalf("taskRecord.toDomain error = %v, want invalid date range", err)
	}
}

func TestScanResultPreservesFailureResultSemantics(t *testing.T) {
	tests := []struct {
		name       string
		status     domain.TaskStatus
		result     domain.SyncResult
		wantResult bool
	}{
		{name: "failed without partial result", status: domain.StatusFailed, result: domain.SyncResult{}, wantResult: false},
		{name: "failed with partial result", status: domain.StatusFailed, result: domain.SyncResult{ProcessedCount: 1}, wantResult: true},
		{name: "succeeded with empty result", status: domain.StatusSucceeded, result: domain.SyncResult{}, wantResult: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := scanResult(test.status, test.result)
			if (got != nil) != test.wantResult {
				t.Fatalf("scanResult present = %t, want %t: %#v", got != nil, test.wantResult, got)
			}
		})
	}
}

func TestPaginationOffsetValidatesPageAndPageSize(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		pageSize int
		want     int
		wantErr  error
	}{
		{name: "first page", page: 1, pageSize: 10, want: 0},
		{name: "second page", page: 2, pageSize: 50, want: 50},
		{name: "page starts at one", page: 0, pageSize: 10, wantErr: domain.ErrInvalidPagination},
		{name: "page size maximum", page: 1, pageSize: 51, wantErr: domain.ErrInvalidPagination},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := paginationOffset(test.page, test.pageSize)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("paginationOffset error = %v, want %v", err, test.wantErr)
			}
			if test.wantErr == nil && got != test.want {
				t.Fatalf("paginationOffset = %d, want %d", got, test.want)
			}
		})
	}
}

func TestDailyBarRecordUsesDateOnlyForPersistence(t *testing.T) {
	value := time.Date(2026, 9, 26, 15, 4, 5, 0, time.FixedZone("CST", 8*60*60))
	record := dailyBarRecordFromDomain(domain.DailyBar{TradeDate: value})
	want := time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC)
	if !record.TradeDate.Equal(want) {
		t.Fatalf("TradeDate = %v, want %v", record.TradeDate, want)
	}
}

func taskRows(now time.Time) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "target", "trigger_type", "status", "active_target", "source_provider", "source_mode",
		"start_date", "end_date", "data_as_of", "created_at", "started_at", "finished_at", "updated_at",
		"next_attempt_at", "retry_count", "max_retries", "failure_reason", "processed_count", "created_count",
		"updated_count", "failed_count", "retry_of_task_id",
	})
}
