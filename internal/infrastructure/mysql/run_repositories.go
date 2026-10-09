package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	backtestdomain "stock-quant/internal/backtest/domain"
	backtestports "stock-quant/internal/backtest/ports"
	marketdomain "stock-quant/internal/market/domain"
	marketports "stock-quant/internal/market/ports"
	screeningdomain "stock-quant/internal/screening/domain"
	screeningports "stock-quant/internal/screening/ports"
	"stock-quant/internal/shared/types"
)

type syncJobRepository struct{ db *sql.DB }
type screeningRunRepository struct{ db *sql.DB }
type backtestRunRepository struct{ db *sql.DB }

func NewSyncJobRepository(db *sql.DB) (marketports.SyncJobStore, error) {
	if db == nil {
		return nil, errors.New("create sync job repository: database is required")
	}
	return &syncJobRepository{db: db}, nil
}
func NewScreeningRunRepository(db *sql.DB) (screeningports.ScreeningStore, error) {
	if db == nil {
		return nil, errors.New("create screening run repository: database is required")
	}
	return &screeningRunRepository{db: db}, nil
}
func NewBacktestRunRepository(db *sql.DB) (backtestports.BacktestStore, error) {
	if db == nil {
		return nil, errors.New("create backtest run repository: database is required")
	}
	return &backtestRunRepository{db: db}, nil
}

var _ marketports.SyncJobStore = (*syncJobRepository)(nil)
var _ screeningports.ScreeningStore = (*screeningRunRepository)(nil)
var _ backtestports.BacktestStore = (*backtestRunRepository)(nil)

func (r *syncJobRepository) CreateOrGet(ctx context.Context, job marketdomain.SyncJob) (marketdomain.SyncJob, bool, error) {
	if strings.TrimSpace(job.JobID) == "" || strings.TrimSpace(job.TaskKey) == "" || strings.TrimSpace(job.APIName) == "" || job.Status != marketdomain.SyncJobPending || job.RequestedAt.IsZero() || job.CreatedAt.IsZero() || job.UpdatedAt.IsZero() {
		return marketdomain.SyncJob{}, false, errors.New("create sync job: id, task key, API, PENDING status, and timestamps are required")
	}
	tradeDate, err := nullableDate(job.TradeDate)
	if err != nil {
		return marketdomain.SyncJob{}, false, err
	}
	var expected any
	if job.ExpectedRows != nil {
		if *job.ExpectedRows < 0 {
			return marketdomain.SyncJob{}, false, errors.New("expected rows cannot be negative")
		}
		expected = *job.ExpectedRows
	}
	if len(job.QualityReport) > 0 && !json.Valid(job.QualityReport) {
		return marketdomain.SyncJob{}, false, errors.New("quality report must be valid JSON")
	}
	result, err := r.db.ExecContext(ctx, `INSERT INTO t_sync_job (job_id,task_key,api_name,trade_date,status,attempts,requested_at,expected_rows,received_rows,quality_report_json,last_error,created_at,updated_at) VALUES (?,?,?,?,?,0,?,?,NULL,NULL,NULL,?,?) ON DUPLICATE KEY UPDATE task_key=t_sync_job.task_key`, job.JobID, job.TaskKey, job.APIName, tradeDate, job.Status, databaseTimestamp(job.RequestedAt), expected, databaseTimestamp(job.CreatedAt), databaseTimestamp(job.UpdatedAt))
	if err != nil {
		return marketdomain.SyncJob{}, false, fmt.Errorf("create or get sync job: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return marketdomain.SyncJob{}, false, fmt.Errorf("read sync job insert result: %w", err)
	}
	created, err := r.findByTaskKey(ctx, job.TaskKey)
	if err != nil {
		return marketdomain.SyncJob{}, false, err
	}
	return created, affected == 1, nil
}

func (r *syncJobRepository) MarkRunning(ctx context.Context, id string, at time.Time) error {
	return guardedTransition(ctx, r.db, "start sync job", `UPDATE t_sync_job SET status='RUNNING',attempts=attempts+1,last_error=NULL,updated_at=? WHERE job_id=? AND status='PENDING'`, []any{databaseTimestamp(at), id}, marketdomain.ErrInvalidSyncJobTransition, func() (string, error) {
		var s string
		err := r.db.QueryRowContext(ctx, "SELECT status FROM t_sync_job WHERE job_id=?", id).Scan(&s)
		return s, err
	})
}
func (r *syncJobRepository) MarkSucceeded(ctx context.Context, id string, rows int, quality json.RawMessage, at time.Time) error {
	if rows < 0 {
		return errors.New("received rows cannot be negative")
	}
	if len(quality) > 0 && !json.Valid(quality) {
		return errors.New("quality report must be valid JSON")
	}
	return guardedTransition(ctx, r.db, "succeed sync job", `UPDATE t_sync_job SET status='SUCCESS',received_rows=?,quality_report_json=?,last_error=NULL,updated_at=? WHERE job_id=? AND status='RUNNING'`, []any{rows, jsonValue(quality), databaseTimestamp(at), id}, marketdomain.ErrInvalidSyncJobTransition, func() (string, error) {
		var s string
		err := r.db.QueryRowContext(ctx, "SELECT status FROM t_sync_job WHERE job_id=?", id).Scan(&s)
		return s, err
	})
}
func (r *syncJobRepository) MarkFailed(ctx context.Context, id, code, message string, at time.Time) error {
	return r.finish(ctx, id, marketdomain.SyncJobFailed, code, message, at)
}
func (r *syncJobRepository) MarkBlocked(ctx context.Context, id, code, message string, at time.Time) error {
	return r.finish(ctx, id, marketdomain.SyncJobBlocked, code, message, at)
}
func (r *syncJobRepository) finish(ctx context.Context, id string, status marketdomain.SyncJobStatus, code, message string, at time.Time) error {
	last := code
	if message != "" {
		if last != "" {
			last += ": "
		}
		last += message
	}
	if len(last) > 1000 {
		last = last[:1000]
	}
	return guardedTransition(ctx, r.db, "finish sync job", `UPDATE t_sync_job SET status=?,last_error=?,updated_at=? WHERE job_id=? AND status='RUNNING'`, []any{status, last, databaseTimestamp(at), id}, marketdomain.ErrInvalidSyncJobTransition, func() (string, error) {
		var s string
		err := r.db.QueryRowContext(ctx, "SELECT status FROM t_sync_job WHERE job_id=?", id).Scan(&s)
		return s, err
	})
}
func (r *syncJobRepository) Find(ctx context.Context, id string) (marketdomain.SyncJob, error) {
	return scanSyncJob(r.db.QueryRowContext(ctx, `SELECT job_id,task_key,api_name,CAST(trade_date AS CHAR),status,attempts,DATE_FORMAT(requested_at,'%Y-%m-%d %H:%i:%s.%f'),expected_rows,received_rows,CAST(quality_report_json AS CHAR),last_error,DATE_FORMAT(created_at,'%Y-%m-%d %H:%i:%s.%f'),DATE_FORMAT(updated_at,'%Y-%m-%d %H:%i:%s.%f') FROM t_sync_job WHERE job_id=?`, id))
}
func (r *syncJobRepository) findByTaskKey(ctx context.Context, key string) (marketdomain.SyncJob, error) {
	return scanSyncJob(r.db.QueryRowContext(ctx, `SELECT job_id,task_key,api_name,CAST(trade_date AS CHAR),status,attempts,DATE_FORMAT(requested_at,'%Y-%m-%d %H:%i:%s.%f'),expected_rows,received_rows,CAST(quality_report_json AS CHAR),last_error,DATE_FORMAT(created_at,'%Y-%m-%d %H:%i:%s.%f'),DATE_FORMAT(updated_at,'%Y-%m-%d %H:%i:%s.%f') FROM t_sync_job WHERE task_key=?`, key))
}

func (r *screeningRunRepository) CreateOrGet(ctx context.Context, run screeningdomain.RunMetadata) (screeningdomain.RunMetadata, bool, error) {
	if err := validateScreeningRun(run); err != nil {
		return screeningdomain.RunMetadata{}, false, err
	}
	result, err := r.db.ExecContext(ctx, `INSERT INTO t_screening_run (run_id,run_key,strategy_id,strategy_version,as_of,config_hash,snapshot_hash,status,total_universe,total_eligible,error_code,error_message,rejection_summary_json,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,0,0,NULL,NULL,NULL,?,?) ON DUPLICATE KEY UPDATE run_key=t_screening_run.run_key`, run.RunID, run.RunKey, run.StrategyID, run.StrategyVersion, run.AsOf.String(), run.ConfigHash, run.SnapshotHash, run.Status, databaseTimestamp(run.CreatedAt), databaseTimestamp(run.UpdatedAt))
	if err != nil {
		return screeningdomain.RunMetadata{}, false, fmt.Errorf("create or get screening run: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return screeningdomain.RunMetadata{}, false, fmt.Errorf("read screening insert result: %w", err)
	}
	got, err := r.findByRunKey(ctx, run.RunKey)
	return got, affected == 1, err
}
func validateScreeningRun(run screeningdomain.RunMetadata) error {
	if strings.TrimSpace(run.RunID) == "" || len(run.RunKey) != 64 || !isHex(run.RunKey) || strings.TrimSpace(run.StrategyID) == "" || strings.TrimSpace(run.StrategyVersion) == "" || !run.AsOf.Valid() || !isHash(run.ConfigHash) || !isHash(run.SnapshotHash) || run.Status != screeningdomain.RunPending || run.CreatedAt.IsZero() || run.UpdatedAt.IsZero() {
		return errors.New("create screening run: valid identity, hashes, PENDING status, and timestamps are required")
	}
	return nil
}
func (r *screeningRunRepository) MarkRunning(ctx context.Context, id string, at time.Time) error {
	return guardedTransition(ctx, r.db, "start screening run", `UPDATE t_screening_run SET status='RUNNING',error_code=NULL,error_message=NULL,updated_at=? WHERE run_id=? AND status='PENDING'`, []any{databaseTimestamp(at), id}, screeningdomain.ErrInvalidStatusTransition, func() (string, error) {
		var s string
		err := r.db.QueryRowContext(ctx, "SELECT status FROM t_screening_run WHERE run_id=?", id).Scan(&s)
		return s, err
	})
}
func (r *screeningRunRepository) MarkFailed(ctx context.Context, id, code, message string, at time.Time) error {
	return r.finish(ctx, id, screeningdomain.RunFailed, code, message, at)
}
func (r *screeningRunRepository) MarkBlocked(ctx context.Context, id, code, message string, at time.Time) error {
	return r.finish(ctx, id, screeningdomain.RunBlocked, code, message, at)
}
func (r *screeningRunRepository) finish(ctx context.Context, id string, status screeningdomain.RunStatus, code, message string, at time.Time) error {
	return guardedTransition(ctx, r.db, "finish screening run", `UPDATE t_screening_run SET status=?,error_code=?,error_message=?,updated_at=? WHERE run_id=? AND status='RUNNING'`, []any{status, nullableString(code), nullableString(message), databaseTimestamp(at), id}, screeningdomain.ErrInvalidStatusTransition, func() (string, error) {
		var s string
		err := r.db.QueryRowContext(ctx, "SELECT status FROM t_screening_run WHERE run_id=?", id).Scan(&s)
		return s, err
	})
}
func (r *screeningRunRepository) Complete(ctx context.Context, id string, outcome screeningdomain.ScreeningOutcome, at time.Time) error {
	if outcome.TotalUniverse < 0 || outcome.TotalEligible < 0 || outcome.TotalEligible > outcome.TotalUniverse {
		return errors.New("screening totals are invalid")
	}
	if len(outcome.RejectionSummary) > 0 && !json.Valid(outcome.RejectionSummary) {
		return errors.New("rejection summary must be valid JSON")
	}
	return inTransaction(ctx, r.db, "complete screening run", func(tx *sql.Tx) error {
		for i, item := range outcome.Results {
			if item.RunID != "" && item.RunID != id {
				return fmt.Errorf("screening result %d belongs to another run", i)
			}
			if strings.TrimSpace(item.TSCode) == "" {
				return fmt.Errorf("screening result %d has no stock code", i)
			}
			if item.FinalRank != nil && *item.FinalRank < 1 {
				return fmt.Errorf("screening result %d has invalid rank", i)
			}
			if item.TotalScore != nil && (*item.TotalScore != *item.TotalScore || *item.TotalScore > 1e308 || *item.TotalScore < -1e308) {
				return fmt.Errorf("screening result %d has invalid score", i)
			}
			for name, value := range map[string]json.RawMessage{"raw factors": item.RawFactors, "factor scores": item.FactorScores, "reasons": item.Reasons} {
				if len(value) > 0 && !json.Valid(value) {
					return fmt.Errorf("screening result %d %s must be valid JSON", i, name)
				}
			}
			var rank, score any
			if item.FinalRank != nil {
				rank = *item.FinalRank
			}
			if item.TotalScore != nil {
				score = *item.TotalScore
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO t_screening_result (run_id,ts_code,final_rank,total_score,raw_factors_json,factor_scores_json,reason_json) VALUES (?,?,?,?,?,?,?)`, id, item.TSCode, rank, score, jsonValue(item.RawFactors), jsonValue(item.FactorScores), jsonValue(item.Reasons)); err != nil {
				return fmt.Errorf("insert screening result %s: %w", item.TSCode, err)
			}
		}
		res, err := tx.ExecContext(ctx, `UPDATE t_screening_run SET status='SUCCESS',total_universe=?,total_eligible=?,rejection_summary_json=?,error_code=NULL,error_message=NULL,updated_at=? WHERE run_id=? AND status='RUNNING'`, outcome.TotalUniverse, outcome.TotalEligible, jsonValue(outcome.RejectionSummary), databaseTimestamp(at), id)
		if err != nil {
			return fmt.Errorf("mark screening success: %w", err)
		}
		return requireOneRow(res, screeningdomain.ErrInvalidStatusTransition, "screening run is not RUNNING")
	})
}
func (r *screeningRunRepository) Find(ctx context.Context, id string) (screeningdomain.RunMetadata, error) {
	return scanScreeningRun(r.db.QueryRowContext(ctx, `SELECT run_id,run_key,CAST(as_of AS CHAR),strategy_id,strategy_version,config_hash,snapshot_hash,status,total_universe,total_eligible,error_code,error_message,CAST(rejection_summary_json AS CHAR),DATE_FORMAT(created_at,'%Y-%m-%d %H:%i:%s.%f'),DATE_FORMAT(updated_at,'%Y-%m-%d %H:%i:%s.%f') FROM t_screening_run WHERE run_id=?`, id))
}
func (r *screeningRunRepository) findByRunKey(ctx context.Context, key string) (screeningdomain.RunMetadata, error) {
	return scanScreeningRun(r.db.QueryRowContext(ctx, `SELECT run_id,run_key,CAST(as_of AS CHAR),strategy_id,strategy_version,config_hash,snapshot_hash,status,total_universe,total_eligible,error_code,error_message,CAST(rejection_summary_json AS CHAR),DATE_FORMAT(created_at,'%Y-%m-%d %H:%i:%s.%f'),DATE_FORMAT(updated_at,'%Y-%m-%d %H:%i:%s.%f') FROM t_screening_run WHERE run_key=?`, key))
}
func (r *screeningRunRepository) ListResults(ctx context.Context, id string, limit, offset int) ([]screeningdomain.ScreeningResult, error) {
	if limit < 1 || limit > 100 || offset < 0 {
		return nil, errors.New("screening result page requires limit 1..100 and nonnegative offset")
	}
	rows, err := r.db.QueryContext(ctx, `SELECT run_id,ts_code,final_rank,total_score,CAST(raw_factors_json AS CHAR),CAST(factor_scores_json AS CHAR),CAST(reason_json AS CHAR) FROM t_screening_result WHERE run_id=? ORDER BY final_rank IS NULL ASC,final_rank ASC,ts_code ASC LIMIT ? OFFSET ?`, id, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list screening results: %w", err)
	}
	defer rows.Close()
	items := []screeningdomain.ScreeningResult{}
	for rows.Next() {
		var x screeningdomain.ScreeningResult
		var rank sql.NullInt64
		var score sql.NullFloat64
		var raw, factors, reasons sql.NullString
		if err := rows.Scan(&x.RunID, &x.TSCode, &rank, &score, &raw, &factors, &reasons); err != nil {
			return nil, fmt.Errorf("scan screening result: %w", err)
		}
		if rank.Valid {
			v := int(rank.Int64)
			x.FinalRank = &v
		}
		if score.Valid {
			v := score.Float64
			x.TotalScore = &v
		}
		x.RawFactors = rawBytes(raw)
		x.FactorScores = rawBytes(factors)
		x.Reasons = rawBytes(reasons)
		items = append(items, x)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read screening results: %w", err)
	}
	return items, nil
}

func (r *backtestRunRepository) CreateOrGet(ctx context.Context, run backtestdomain.Run) (backtestdomain.Run, bool, error) {
	canonical := backtestdomain.CanonicalRunKey(run)
	if run.RunKey != "" && run.RunKey != canonical {
		return backtestdomain.Run{}, false, errors.New("backtest run key does not match immutable inputs")
	}
	run.RunKey = canonical
	if strings.TrimSpace(run.RunID) == "" || strings.TrimSpace(run.StrategyID) == "" || strings.TrimSpace(run.StrategyVersion) == "" || !run.StartDate.Valid() || !run.EndDate.Valid() || run.StartDate.String() > run.EndDate.String() || !isHash(run.ConfigHash) || !isHash(run.SnapshotHash) || strings.TrimSpace(run.Mode) == "" || run.Status != backtestdomain.RunPending || run.CreatedAt.IsZero() || run.UpdatedAt.IsZero() {
		return backtestdomain.Run{}, false, errors.New("create backtest run: valid identity, range, hashes, PENDING status, and timestamps are required")
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO t_backtest_run (run_id,run_key,strategy_id,strategy_version,start_date,end_date,config_hash,snapshot_hash,mode,status,metrics_json,risk_json,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,NULL,NULL,?,?) ON DUPLICATE KEY UPDATE run_key=t_backtest_run.run_key`, run.RunID, run.RunKey, run.StrategyID, run.StrategyVersion, run.StartDate.String(), run.EndDate.String(), run.ConfigHash, run.SnapshotHash, run.Mode, run.Status, databaseTimestamp(run.CreatedAt), databaseTimestamp(run.UpdatedAt))
	if err != nil {
		return backtestdomain.Run{}, false, fmt.Errorf("create or get backtest run: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return backtestdomain.Run{}, false, fmt.Errorf("read backtest insert result: %w", err)
	}
	got, err := r.findByRunKey(ctx, run.RunKey)
	return got, affected == 1, err
}
func (r *backtestRunRepository) MarkRunning(ctx context.Context, id string, at time.Time) error {
	return guardedTransition(ctx, r.db, "start backtest run", `UPDATE t_backtest_run SET status='RUNNING',updated_at=? WHERE run_id=? AND status='PENDING'`, []any{databaseTimestamp(at), id}, backtestdomain.ErrInvalidStatusTransition, func() (string, error) {
		var s string
		err := r.db.QueryRowContext(ctx, "SELECT status FROM t_backtest_run WHERE run_id=?", id).Scan(&s)
		return s, err
	})
}
func (r *backtestRunRepository) MarkFailed(ctx context.Context, id, code, message string, at time.Time) error {
	return r.finish(ctx, id, backtestdomain.RunFailed, code, message, at)
}
func (r *backtestRunRepository) MarkBlocked(ctx context.Context, id, code, message string, at time.Time) error {
	return r.finish(ctx, id, backtestdomain.RunBlocked, code, message, at)
}
func (r *backtestRunRepository) finish(ctx context.Context, id string, status backtestdomain.RunStatus, code, message string, at time.Time) error {
	last := code
	if message != "" {
		if last != "" {
			last += ": "
		}
		last += message
	}
	if len(last) > 1000 {
		last = last[:1000]
	}
	return guardedTransition(ctx, r.db, "finish backtest run", `UPDATE t_backtest_run SET status=?,error_message=?,updated_at=? WHERE run_id=? AND status='RUNNING'`, []any{status, nullableString(last), databaseTimestamp(at), id}, backtestdomain.ErrInvalidStatusTransition, func() (string, error) {
		var s string
		err := r.db.QueryRowContext(ctx, "SELECT status FROM t_backtest_run WHERE run_id=?", id).Scan(&s)
		return s, err
	})
}
func (r *backtestRunRepository) Complete(ctx context.Context, id string, metrics, risk []byte, at time.Time) error {
	if len(metrics) > 0 && !json.Valid(metrics) {
		return errors.New("backtest metrics must be valid JSON")
	}
	if len(risk) > 0 && !json.Valid(risk) {
		return errors.New("backtest risk must be valid JSON")
	}
	return guardedTransition(ctx, r.db, "complete backtest run", `UPDATE t_backtest_run SET status='SUCCESS',metrics_json=?,risk_json=?,error_message=NULL,updated_at=? WHERE run_id=? AND status='RUNNING'`, []any{jsonValue(metrics), jsonValue(risk), databaseTimestamp(at), id}, backtestdomain.ErrInvalidStatusTransition, func() (string, error) {
		var s string
		err := r.db.QueryRowContext(ctx, "SELECT status FROM t_backtest_run WHERE run_id=?", id).Scan(&s)
		return s, err
	})
}
func (r *backtestRunRepository) Find(ctx context.Context, id string) (backtestdomain.Run, error) {
	return scanBacktestRun(r.db.QueryRowContext(ctx, `SELECT run_id,run_key,strategy_id,strategy_version,CAST(start_date AS CHAR),CAST(end_date AS CHAR),config_hash,snapshot_hash,mode,status,CAST(metrics_json AS CHAR),CAST(risk_json AS CHAR),error_message,DATE_FORMAT(created_at,'%Y-%m-%d %H:%i:%s.%f'),DATE_FORMAT(updated_at,'%Y-%m-%d %H:%i:%s.%f') FROM t_backtest_run WHERE run_id=?`, id))
}
func (r *backtestRunRepository) findByRunKey(ctx context.Context, key string) (backtestdomain.Run, error) {
	return scanBacktestRun(r.db.QueryRowContext(ctx, `SELECT run_id,run_key,strategy_id,strategy_version,CAST(start_date AS CHAR),CAST(end_date AS CHAR),config_hash,snapshot_hash,mode,status,CAST(metrics_json AS CHAR),CAST(risk_json AS CHAR),error_message,DATE_FORMAT(created_at,'%Y-%m-%d %H:%i:%s.%f'),DATE_FORMAT(updated_at,'%Y-%m-%d %H:%i:%s.%f') FROM t_backtest_run WHERE run_key=?`, key))
}

type rowScanner interface{ Scan(...any) error }

func scanSyncJob(row rowScanner) (marketdomain.SyncJob, error) {
	var x marketdomain.SyncJob
	var date, requested, created, updated, last sql.NullString
	var expected, received sql.NullInt64
	var quality sql.NullString
	var status string
	if err := row.Scan(&x.JobID, &x.TaskKey, &x.APIName, &date, &status, &x.Attempts, &requested, &expected, &received, &quality, &last, &created, &updated); err != nil {
		return x, fmt.Errorf("scan sync job: %w", err)
	}
	x.Status = marketdomain.SyncJobStatus(status)
	if date.Valid {
		d, e := types.ParseTradingDate(date.String)
		if e != nil {
			return x, e
		}
		x.TradeDate = &d
	}
	if expected.Valid {
		v := int(expected.Int64)
		x.ExpectedRows = &v
	}
	if received.Valid {
		v := int(received.Int64)
		x.ReceivedRows = &v
	}
	x.QualityReport = rawBytes(quality)
	if last.Valid {
		x.LastError = &last.String
	}
	var err error
	if x.RequestedAt, err = parseDatabaseTimestamp(requested.String); err != nil {
		return x, err
	}
	if x.CreatedAt, err = parseDatabaseTimestamp(created.String); err != nil {
		return x, err
	}
	x.UpdatedAt, err = parseDatabaseTimestamp(updated.String)
	return x, err
}
func scanScreeningRun(row rowScanner) (screeningdomain.RunMetadata, error) {
	var x screeningdomain.RunMetadata
	var date, code, message, rejection sql.NullString
	var created, updated string
	var status string
	if err := row.Scan(&x.RunID, &x.RunKey, &date, &x.StrategyID, &x.StrategyVersion, &x.ConfigHash, &x.SnapshotHash, &status, &x.TotalUniverse, &x.TotalEligible, &code, &message, &rejection, &created, &updated); err != nil {
		return x, fmt.Errorf("scan screening run: %w", err)
	}
	var err error
	x.AsOf, err = types.ParseTradingDate(date.String)
	if err != nil {
		return x, err
	}
	x.Status = screeningdomain.RunStatus(status)
	if code.Valid {
		x.ErrorCode = &code.String
	}
	if message.Valid {
		x.ErrorMessage = &message.String
	}
	x.RejectionSummary = rawBytes(rejection)
	if x.CreatedAt, err = parseDatabaseTimestamp(created); err != nil {
		return x, err
	}
	x.UpdatedAt, err = parseDatabaseTimestamp(updated)
	return x, err
}
func scanBacktestRun(row rowScanner) (backtestdomain.Run, error) {
	var x backtestdomain.Run
	var start, end string
	var status string
	var snapshot, metrics, risk, failure sql.NullString
	var created, updated string
	if err := row.Scan(&x.RunID, &x.RunKey, &x.StrategyID, &x.StrategyVersion, &start, &end, &x.ConfigHash, &snapshot, &x.Mode, &status, &metrics, &risk, &failure, &created, &updated); err != nil {
		return x, fmt.Errorf("scan backtest run: %w", err)
	}
	if snapshot.Valid {
		x.SnapshotHash = snapshot.String
	}
	var err error
	x.StartDate, err = types.ParseTradingDate(start)
	if err != nil {
		return x, err
	}
	x.EndDate, err = types.ParseTradingDate(end)
	if err != nil {
		return x, err
	}
	x.Status = backtestdomain.RunStatus(status)
	x.Metrics = rawBytes(metrics)
	x.Risk = rawBytes(risk)
	if failure.Valid {
		x.ErrorMessage = &failure.String
	}
	if x.CreatedAt, err = parseDatabaseTimestamp(created); err != nil {
		return x, err
	}
	x.UpdatedAt, err = parseDatabaseTimestamp(updated)
	return x, err
}

func guardedTransition(ctx context.Context, db *sql.DB, operation, query string, args []any, sentinel error, current func() (string, error)) error {
	res, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	if err = requireOneRow(res, sentinel, "state transition was rejected"); err == nil {
		return nil
	}
	status, readErr := current()
	if readErr != nil {
		return errors.Join(err, fmt.Errorf("read current state: %w", readErr))
	}
	if status == "" {
		return fmt.Errorf("%s: record not found", operation)
	}
	return fmt.Errorf("%s: current status %s: %w", operation, status, sentinel)
}
func requireOneRow(res sql.Result, sentinel error, message string) error {
	count, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if count != 1 {
		return fmt.Errorf("%s: %w", message, sentinel)
	}
	return nil
}
func jsonValue(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return string(value)
}
func rawBytes(value sql.NullString) json.RawMessage {
	if !value.Valid {
		return nil
	}
	return json.RawMessage(value.String)
}
func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
func isHex(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F') {
			return false
		}
	}
	return true
}
func isHash(value string) bool { return isHex(value) }
