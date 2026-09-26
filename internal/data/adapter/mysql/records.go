package mysql

import (
	"fmt"
	"time"

	"github.com/disturb-yy/stock-quant/internal/data/domain"
)

type instrumentRecord struct {
	ID             uint64
	Symbol         string
	Market         string
	Name           string
	Status         string
	SourceProvider string
	SourceMode     string
	DataAsOf       *time.Time
	UpdatedAt      time.Time
}

func (instrumentRecord) TableName() string {
	return "t_instrument"
}

func instrumentRecordFromDomain(instrument domain.Instrument) instrumentRecord {
	return instrumentRecord{
		Symbol:         instrument.Symbol,
		Market:         instrument.Market,
		Name:           instrument.Name,
		Status:         instrument.Status,
		SourceProvider: instrument.Source.Provider,
		SourceMode:     instrument.Source.Mode,
		DataAsOf:       instrument.DataAsOf,
		UpdatedAt:      instrument.UpdatedAt,
	}
}

func (record instrumentRecord) toDomain() (domain.Instrument, error) {
	if record.UpdatedAt.IsZero() {
		return domain.Instrument{}, fmt.Errorf("instrument updated_at is empty")
	}
	return domain.Instrument{
		Symbol:    record.Symbol,
		Market:    record.Market,
		Name:      record.Name,
		Status:    record.Status,
		Source:    domain.SourceProvenance{Provider: record.SourceProvider, Mode: record.SourceMode},
		DataAsOf:  datePointer(record.DataAsOf),
		UpdatedAt: record.UpdatedAt,
	}, nil
}

type dailyBarRecord struct {
	ID             uint64
	Symbol         string
	Market         string
	TradeDate      time.Time
	OpenPrice      float64
	HighPrice      float64
	LowPrice       float64
	ClosePrice     float64
	Volume         float64
	SourceProvider string
	SourceMode     string
	UpdatedAt      time.Time
}

func (dailyBarRecord) TableName() string {
	return "t_instrument_daily_bar"
}

func dailyBarRecordFromDomain(bar domain.DailyBar) dailyBarRecord {
	return dailyBarRecord{
		Symbol:         bar.Symbol,
		Market:         bar.Market,
		TradeDate:      dateTime(bar.TradeDate),
		OpenPrice:      bar.Open,
		HighPrice:      bar.High,
		LowPrice:       bar.Low,
		ClosePrice:     bar.Close,
		Volume:         bar.Volume,
		SourceProvider: bar.Source.Provider,
		SourceMode:     bar.Source.Mode,
		UpdatedAt:      bar.UpdatedAt,
	}
}

func (record dailyBarRecord) toDomain() (domain.DailyBar, error) {
	if record.TradeDate.IsZero() || record.UpdatedAt.IsZero() {
		return domain.DailyBar{}, fmt.Errorf("daily bar date or updated_at is empty")
	}
	return domain.DailyBar{
		Symbol:    record.Symbol,
		Market:    record.Market,
		TradeDate: dateTime(record.TradeDate),
		Open:      record.OpenPrice,
		High:      record.HighPrice,
		Low:       record.LowPrice,
		Close:     record.ClosePrice,
		Volume:    record.Volume,
		Source:    domain.SourceProvenance{Provider: record.SourceProvider, Mode: record.SourceMode},
		UpdatedAt: record.UpdatedAt,
	}, nil
}

type taskRecord struct {
	ID             string
	Target         string
	TriggerType    string
	Status         string
	ActiveTarget   *string
	SourceProvider string
	SourceMode     string
	StartDate      *time.Time
	EndDate        *time.Time
	DataAsOf       *time.Time
	CreatedAt      time.Time
	StartedAt      *time.Time
	FinishedAt     *time.Time
	UpdatedAt      time.Time
	NextAttemptAt  *time.Time
	RetryCount     int
	MaxRetries     int
	FailureReason  *string
	ProcessedCount int
	CreatedCount   int
	UpdatedCount   int
	FailedCount    int
	RetryOfTaskID  *string
}

func (taskRecord) TableName() string {
	return "t_data_sync_task"
}

func taskRecordFromDomain(task domain.SyncTask) taskRecord {
	return taskRecord{
		ID:             task.ID,
		Target:         string(task.Target),
		TriggerType:    string(task.Trigger),
		Status:         string(task.Status),
		ActiveTarget:   stringPointer(string(task.Target)),
		SourceProvider: task.Source.Provider,
		SourceMode:     task.Source.Mode,
		StartDate:      datePointer(task.DateRange.Start),
		EndDate:        datePointer(task.DateRange.End),
		DataAsOf:       datePointer(task.DataAsOf),
		CreatedAt:      task.CreatedAt,
		StartedAt:      task.StartedAt,
		FinishedAt:     task.FinishedAt,
		UpdatedAt:      task.UpdatedAt,
		NextAttemptAt:  task.NextAttemptAt,
		RetryCount:     task.RetryCount,
		MaxRetries:     task.MaxRetries,
		FailureReason:  stringPointer(task.FailureReason),
		RetryOfTaskID:  task.RetryOfTaskID,
	}
}

func (record taskRecord) toDomain() (domain.SyncTask, error) {
	dateRange, err := parseDateRange(record.StartDate, record.EndDate)
	if err != nil {
		return domain.SyncTask{}, err
	}
	if record.CreatedAt.IsZero() || record.UpdatedAt.IsZero() {
		return domain.SyncTask{}, fmt.Errorf("sync task timestamps are empty")
	}
	task := domain.SyncTask{
		ID:            record.ID,
		Target:        domain.SyncTarget(record.Target),
		Trigger:       domain.Trigger(record.TriggerType),
		Status:        domain.TaskStatus(record.Status),
		Source:        domain.SourceProvenance{Provider: record.SourceProvider, Mode: record.SourceMode},
		DateRange:     dateRange,
		CreatedAt:     record.CreatedAt,
		StartedAt:     timePointer(record.StartedAt),
		FinishedAt:    timePointer(record.FinishedAt),
		UpdatedAt:     record.UpdatedAt,
		DataAsOf:      datePointer(record.DataAsOf),
		RetryCount:    record.RetryCount,
		MaxRetries:    record.MaxRetries,
		RetryOfTaskID: stringValuePointer(record.RetryOfTaskID),
	}
	if record.FailureReason != nil {
		task.FailureReason = *record.FailureReason
	}
	task.NextAttemptAt = timePointer(record.NextAttemptAt)
	task.Result = scanResult(task.Status, domain.SyncResult{
		ProcessedCount: record.ProcessedCount,
		CreatedCount:   record.CreatedCount,
		UpdatedCount:   record.UpdatedCount,
		FailedCount:    record.FailedCount,
	})
	return task, nil
}

func parseDateRange(start, end *time.Time) (domain.DateRange, error) {
	if start == nil && end == nil {
		return domain.DateRange{}, nil
	}
	if start == nil || end == nil {
		return domain.DateRange{}, domain.ErrInvalidDateRange
	}
	startDate := dateTime(*start)
	endDate := dateTime(*end)
	return domain.NewDateRange(&startDate, &endDate)
}

func datePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	date := dateTime(*value)
	return &date
}

func timePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func stringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func stringValuePointer(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func dateTime(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}
