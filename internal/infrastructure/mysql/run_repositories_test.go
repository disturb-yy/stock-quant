package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	backtestdomain "stock-quant/internal/backtest/domain"
	marketdomain "stock-quant/internal/market/domain"
	screeningdomain "stock-quant/internal/screening/domain"
	"stock-quant/internal/shared/types"
)

func TestScreeningCreateOrGetConcurrentRunKeyMySQL(t *testing.T) {
	db := openIsolatedMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	migrateForRepositoryTest(t, ctx, db)
	repository, err := NewScreeningRunRepository(db)
	if err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	type result struct {
		run     screeningdomain.RunMetadata
		created bool
		err     error
	}
	results := make(chan result, 2)
	for i := 1; i <= 2; i++ {
		run := screeningRun(i, strings.Repeat("a", 64))
		go func(run screeningdomain.RunMetadata) {
			<-start
			got, created, err := repository.CreateOrGet(ctx, run)
			results <- result{run: got, created: created, err: err}
		}(run)
	}
	close(start)
	createdCount := 0
	var winner string
	for range 2 {
		got := <-results
		if got.err != nil {
			t.Fatalf("CreateOrGet() error = %v", got.err)
		}
		if got.created {
			createdCount++
		}
		if winner == "" {
			winner = got.run.RunID
		} else if got.run.RunID != winner {
			t.Fatalf("concurrent callers received different run ids: %q and %q", winner, got.run.RunID)
		}
	}
	if createdCount != 1 {
		t.Fatalf("new runs created = %d, want exactly 1", createdCount)
	}
	var rows int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM t_screening_run WHERE run_key = ?", strings.Repeat("a", 64)).Scan(&rows); err != nil || rows != 1 {
		t.Fatalf("screening rows for duplicate key = %d, error %v; want 1", rows, err)
	}
}

func TestScreeningCompleteRollsBackWhenResultInsertFailsMySQL(t *testing.T) {
	db := openIsolatedMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	migrateForRepositoryTest(t, ctx, db)
	repository, err := NewScreeningRunRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	run := screeningRun(3, strings.Repeat("b", 64))
	if _, _, err := repository.CreateOrGet(ctx, run); err != nil {
		t.Fatal(err)
	}
	if err := repository.MarkRunning(ctx, run.RunID, fixedFetchedAt()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TRIGGER fail_second_screening_result BEFORE INSERT ON t_screening_result FOR EACH ROW
BEGIN
  IF NEW.ts_code = '000002.SZ' THEN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'injected result insert failure';
  END IF;
END`); err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}
	outcome := screeningdomain.ScreeningOutcome{
		TotalUniverse: 2, TotalEligible: 2, RejectionSummary: json.RawMessage(`{"below_threshold":0}`),
		Results: []screeningdomain.ScreeningResult{
			screeningResult(run.RunID, "000001.SZ", 1),
			screeningResult(run.RunID, "000002.SZ", 2),
		},
	}
	if err := repository.Complete(ctx, run.RunID, outcome, fixedFetchedAt()); err == nil || !strings.Contains(err.Error(), "injected result insert failure") {
		t.Fatalf("Complete() error = %v; want injected SQL failure", err)
	}
	got, err := repository.Find(ctx, run.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != screeningdomain.RunRunning {
		t.Fatalf("status after rollback = %q, want RUNNING", got.Status)
	}
	results, err := repository.ListResults(ctx, run.RunID, 10, 0)
	if err != nil || len(results) != 0 {
		t.Fatalf("results after rollback = %#v, error %v; want empty", results, err)
	}
}

func TestScreeningResultPaginationAndSuccessTerminalMySQL(t *testing.T) {
	db := openIsolatedMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	migrateForRepositoryTest(t, ctx, db)
	repository, err := NewScreeningRunRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	run := screeningRun(4, strings.Repeat("c", 64))
	if _, _, err := repository.CreateOrGet(ctx, run); err != nil {
		t.Fatal(err)
	}
	if err := repository.MarkRunning(ctx, run.RunID, fixedFetchedAt()); err != nil {
		t.Fatal(err)
	}
	unranked := screeningResult(run.RunID, "000004.SZ", 0)
	unranked.FinalRank = nil
	outcome := screeningdomain.ScreeningOutcome{
		TotalUniverse: 4, TotalEligible: 3, RejectionSummary: json.RawMessage(`{"filtered":1}`),
		Results: []screeningdomain.ScreeningResult{
			screeningResult(run.RunID, "000002.SZ", 1),
			screeningResult(run.RunID, "000003.SZ", 2),
			screeningResult(run.RunID, "000001.SZ", 1),
			unranked,
		},
	}
	if err := repository.Complete(ctx, run.RunID, outcome, fixedFetchedAt()); err != nil {
		t.Fatal(err)
	}
	firstPage, err := repository.ListResults(ctx, run.RunID, 2, 0)
	if err != nil || len(firstPage) != 2 || firstPage[0].TSCode != "000001.SZ" || firstPage[1].TSCode != "000002.SZ" {
		t.Fatalf("first page = %#v, error %v; want rank/code order", firstPage, err)
	}
	secondPage, err := repository.ListResults(ctx, run.RunID, 2, 2)
	if err != nil || len(secondPage) != 2 || secondPage[0].TSCode != "000003.SZ" || secondPage[1].TSCode != "000004.SZ" {
		t.Fatalf("second page = %#v, error %v; want rank then unranked", secondPage, err)
	}
	if len(secondPage[0].RawFactors) == 0 || len(secondPage[0].Reasons) == 0 {
		t.Fatalf("result lost factor/reason JSON: %#v", secondPage[0])
	}
	if err := repository.MarkFailed(ctx, run.RunID, "FAILED_LATE", "late error", fixedFetchedAt()); !errors.Is(err, screeningdomain.ErrInvalidStatusTransition) {
		t.Fatalf("MarkFailed() after success = %v; want invalid transition", err)
	}
	got, err := repository.Find(ctx, run.RunID)
	if err != nil || got.Status != screeningdomain.RunSuccess || got.SnapshotHash != run.SnapshotHash {
		t.Fatalf("run after late failure = %#v, error %v", got, err)
	}
}

func TestSyncJobIdempotencyStatusAndAttemptsMySQL(t *testing.T) {
	db := openIsolatedMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	migrateForRepositoryTest(t, ctx, db)
	repository, err := NewSyncJobRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	job := syncJob(1, "daily/SSE/2026-06-04")
	created, inserted, err := repository.CreateOrGet(ctx, job)
	if err != nil || !inserted || created.JobID != job.JobID {
		t.Fatalf("first CreateOrGet() = %#v, %v, %v", created, inserted, err)
	}
	duplicate := job
	duplicate.JobID = idempotentID(2)
	duplicate.APIName = "not-used-for-existing-key"
	got, inserted, err := repository.CreateOrGet(ctx, duplicate)
	if err != nil || inserted || got.JobID != job.JobID || got.APIName != job.APIName {
		t.Fatalf("duplicate CreateOrGet() = %#v, %v, %v; want original job", got, inserted, err)
	}
	if err := repository.MarkRunning(ctx, job.JobID, fixedFetchedAt()); err != nil {
		t.Fatal(err)
	}
	quality := json.RawMessage(`{"valid":true}`)
	if err := repository.MarkSucceeded(ctx, job.JobID, 4, quality, fixedFetchedAt().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := repository.MarkFailed(ctx, job.JobID, "LATE", "late failure", fixedFetchedAt()); !errors.Is(err, marketdomain.ErrInvalidSyncJobTransition) {
		t.Fatalf("MarkFailed() after success = %v; want invalid transition", err)
	}
	got, err = repository.Find(ctx, job.JobID)
	if err != nil || got.Status != marketdomain.SyncJobSuccess || got.Attempts != 1 || got.ReceivedRows == nil || *got.ReceivedRows != 4 {
		t.Fatalf("sync job after completion = %#v, error %v", got, err)
	}
}

func TestBacktestCreateOrGetBindsSnapshotAndPersistsOutcomeMySQL(t *testing.T) {
	db := openIsolatedMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	migrateForRepositoryTest(t, ctx, db)
	repository, err := NewBacktestRunRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	run := backtestRun(1, strings.Repeat("d", 64))
	created, inserted, err := repository.CreateOrGet(ctx, run)
	if err != nil || !inserted || created.RunKey != run.RunKey {
		t.Fatalf("first CreateOrGet() = %#v, %v, %v", created, inserted, err)
	}
	duplicate := run
	duplicate.RunID = idempotentID(2)
	got, inserted, err := repository.CreateOrGet(ctx, duplicate)
	if err != nil || inserted || got.RunID != run.RunID {
		t.Fatalf("duplicate CreateOrGet() = %#v, %v, %v", got, inserted, err)
	}
	otherSnapshot := backtestRun(3, strings.Repeat("e", 64))
	if otherSnapshot.RunKey == run.RunKey {
		t.Fatal("snapshot hash must contribute to the canonical backtest key")
	}
	if _, inserted, err := repository.CreateOrGet(ctx, otherSnapshot); err != nil || !inserted {
		t.Fatalf("run with a different snapshot was not independently created: inserted=%v err=%v", inserted, err)
	}
	if err := repository.MarkRunning(ctx, run.RunID, fixedFetchedAt()); err != nil {
		t.Fatal(err)
	}
	if err := repository.Complete(ctx, run.RunID, []byte(`{"return":0.12}`), []byte(`{"max_drawdown":0.08}`), fixedFetchedAt()); err != nil {
		t.Fatal(err)
	}
	if err := repository.MarkFailed(ctx, run.RunID, "LATE", "late failure", fixedFetchedAt()); !errors.Is(err, backtestdomain.ErrInvalidStatusTransition) {
		t.Fatalf("MarkFailed() after success = %v; want invalid transition", err)
	}
	got, err = repository.Find(ctx, run.RunID)
	var metrics map[string]float64
	if err == nil {
		err = json.Unmarshal(got.Metrics, &metrics)
	}
	if err != nil || got.Status != backtestdomain.RunSuccess || got.SnapshotHash != run.SnapshotHash || metrics["return"] != 0.12 {
		t.Fatalf("backtest after completion = %#v, error %v", got, err)
	}
}

func TestRunRepositoriesReturnDatabaseFailure(t *testing.T) {
	db, err := sql.Open("mysql", "root@unix(/tmp/stock-quant-missing-run-repository.sock)/stock_quant")
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	ctx := context.Background()
	jobRepo, _ := NewSyncJobRepository(db)
	if _, _, err := jobRepo.CreateOrGet(ctx, syncJob(1, "daily/SSE/2026-06-04")); err == nil || !strings.Contains(err.Error(), "database is closed") {
		t.Fatalf("SyncJobStore.CreateOrGet() error = %v, want closed database", err)
	}
	screenRepo, _ := NewScreeningRunRepository(db)
	if _, _, err := screenRepo.CreateOrGet(ctx, screeningRun(1, strings.Repeat("f", 64))); err == nil || !strings.Contains(err.Error(), "database is closed") {
		t.Fatalf("ScreeningStore.CreateOrGet() error = %v, want closed database", err)
	}
	backtestRepo, _ := NewBacktestRunRepository(db)
	if _, _, err := backtestRepo.CreateOrGet(ctx, backtestRun(1, strings.Repeat("a", 64))); err == nil || !strings.Contains(err.Error(), "database is closed") {
		t.Fatalf("BacktestStore.CreateOrGet() error = %v, want closed database", err)
	}
}

func screeningRun(index int, key string) screeningdomain.RunMetadata {
	date := tradingDateForRun("2026-06-04")
	return screeningdomain.RunMetadata{
		RunID: idempotentID(index), RunKey: key, AsOf: date, StrategyID: "momentum_v1", StrategyVersion: "1.0.0",
		ConfigHash: strings.Repeat("1", 64), SnapshotHash: strings.Repeat("2", 64), Status: screeningdomain.RunPending,
		CreatedAt: fixedFetchedAt(), UpdatedAt: fixedFetchedAt(),
	}
}

func screeningResult(runID, code string, rank int) screeningdomain.ScreeningResult {
	return screeningdomain.ScreeningResult{
		RunID: runID, TSCode: code, FinalRank: &rank, TotalScore: floatPointer(75),
		RawFactors: json.RawMessage(`{"momentum_20":0.07}`), FactorScores: json.RawMessage(`{"momentum_20":60}`),
		Reasons: json.RawMessage(`{"accepted":["CLOSE_GT_MA20"]}`),
	}
}

func syncJob(index int, key string) marketdomain.SyncJob {
	date := tradingDateForRun("2026-06-04")
	return marketdomain.SyncJob{
		JobID: idempotentID(index), TaskKey: key, APIName: "daily", TradeDate: &date,
		Status: marketdomain.SyncJobPending, RequestedAt: fixedFetchedAt(), CreatedAt: fixedFetchedAt(), UpdatedAt: fixedFetchedAt(),
	}
}

func backtestRun(index int, snapshotHash string) backtestdomain.Run {
	run := backtestdomain.Run{
		RunID: idempotentID(index), StrategyID: "momentum_v1", StrategyVersion: "1.0.0",
		StartDate: tradingDateForRun("2026-01-01"), EndDate: tradingDateForRun("2026-06-04"),
		ConfigHash: strings.Repeat("3", 64), SnapshotHash: snapshotHash, Mode: "research_only",
		Status: backtestdomain.RunPending, CreatedAt: fixedFetchedAt(), UpdatedAt: fixedFetchedAt(),
	}
	run.RunKey = backtestdomain.CanonicalRunKey(run)
	return run
}

func idempotentID(index int) string { return fmt.Sprintf("0000000000000000000000%04d", index) }

func tradingDateForRun(value string) types.TradingDate {
	date, err := types.ParseTradingDate(value)
	if err != nil {
		panic(err)
	}
	return date
}

func floatPointer(value float64) *float64 { return &value }
