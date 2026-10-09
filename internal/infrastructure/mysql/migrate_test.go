package mysql

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"database/sql"

	migrations "stock-quant/db/migrations"

	mysqldriver "github.com/go-sql-driver/mysql"
)

func TestMigrateUpDownRepeatMySQL(t *testing.T) {
	db := openIsolatedMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	unrelated := "CREATE TABLE unrelated_fixture (id INT PRIMARY KEY)"
	if _, err := db.ExecContext(ctx, unrelated); err != nil {
		t.Fatalf("create unrelated fixture: %v", err)
	}

	migrator := newEmbeddedMigrator(t, db)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("first up: %v", err)
	}
	if got := countTables(t, ctx, db); got != 15 {
		t.Fatalf("table count after up = %d, want 15 including ledger and unrelated fixture", got)
	}
	assertTablesExist(t, ctx, db, []string{
		"t_stock", "t_trade_calendar", "t_daily_price", "t_adj_factor", "t_stock_status_daily",
		"t_sync_job", "t_data_snapshot", "t_strategy", "t_screening_run", "t_screening_result",
		"t_backtest_run", "t_backtest_equity", "t_backtest_trade", "t_schema_migrations",
	})
	assertIndexExists(t, ctx, db, "t_daily_price", "idx_daily_by_date")
	assertMigrationState(t, ctx, db, 1, false, "up")
	assertMigrationState(t, ctx, db, 2, false, "up")
	assertColumnExists(t, ctx, db, "t_backtest_run", "run_key")
	assertColumnExists(t, ctx, db, "t_backtest_run", "snapshot_hash")
	assertIndexExists(t, ctx, db, "t_backtest_run", "uq_backtest_run_key")

	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("repeated up: %v", err)
	}
	if got := countTables(t, ctx, db); got != 15 {
		t.Fatalf("table count after repeated up = %d, want 15", got)
	}

	if err := migrator.Down(ctx); err != nil {
		t.Fatalf("down: %v", err)
	}
	if got := countTables(t, ctx, db); got != 15 {
		t.Fatalf("table count after rolling back migration 2 = %d, want migration 1 tables, ledger, and unrelated fixture", got)
	}
	if exists := tableExists(t, ctx, db, "unrelated_fixture"); !exists {
		t.Fatal("down removed a table not created by the migration")
	}
	if exists := tableExists(t, ctx, db, "t_stock"); !exists {
		t.Fatal("rolling back migration 2 removed a table from migration 1")
	}
	if exists := columnExists(t, ctx, db, "t_backtest_run", "run_key"); exists {
		t.Fatal("migration 2 down left run_key")
	}
	if err := migrator.Down(ctx); err != nil {
		t.Fatalf("down migration 1: %v", err)
	}
	if got := countTables(t, ctx, db); got != 2 {
		t.Fatalf("table count after all down migrations = %d, want unrelated fixture and migration ledger", got)
	}

	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("up after down: %v", err)
	}
	assertMigrationState(t, ctx, db, 2, false, "up")
}

func TestMigrateFailedUpCanResumeMySQL(t *testing.T) {
	db := openIsolatedMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	migration := Migration{
		Version: 1,
		Name:    "recovery fixture",
		Up:      "CREATE TABLE IF NOT EXISTS t_recovery_base (id INT PRIMARY KEY); CREATE INDEX idx_recovery ON t_recovery_base (later_column);",
		Down:    "DROP TABLE IF EXISTS t_recovery_base;",
	}
	migrator := &Migrator{DB: db, Migrations: []Migration{migration}}
	if err := migrator.Up(ctx); err == nil {
		t.Fatal("up with conflicting table unexpectedly succeeded")
	}
	assertMigrationState(t, ctx, db, 1, true, "up")
	if !tableExists(t, ctx, db, "t_recovery_base") {
		t.Fatal("first DDL statement was not retained after the later statement failed")
	}

	if _, err := db.ExecContext(ctx, "ALTER TABLE t_recovery_base ADD COLUMN later_column INT NULL"); err != nil {
		t.Fatalf("repair partial migration fixture: %v", err)
	}
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("retry dirty migration: %v", err)
	}
	assertMigrationState(t, ctx, db, 1, false, "up")
	var indexes int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 't_recovery_base' AND index_name = 'idx_recovery'").Scan(&indexes); err != nil || indexes != 1 {
		t.Fatalf("recovered migration index count = %d, %v; want 1", indexes, err)
	}
}

func TestBacktestIdentityMigrationPreservesLegacyRowsMySQL(t *testing.T) {
	db := openIsolatedMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	migrationsToApply, err := LoadMigrations(migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	migrationOne := &Migrator{DB: db, Migrations: migrationsToApply[:1]}
	if err := migrationOne.Up(ctx); err != nil {
		t.Fatalf("apply migration 1: %v", err)
	}
	const legacyID = "00000000000000000000000001"
	if _, err := db.ExecContext(ctx, `INSERT INTO t_backtest_run (run_id,strategy_id,strategy_version,start_date,end_date,config_hash,mode,status,metrics_json,risk_json,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, legacyID, "legacy", "1.0", "2026-01-01", "2026-01-02", strings.Repeat("a", 64), "research_only", "SUCCESS", `{"return":0.1}`, nil, "2026-01-02 00:00:00", "2026-01-02 00:00:00"); err != nil {
		t.Fatalf("insert legacy backtest row: %v", err)
	}
	if err := newEmbeddedMigrator(t, db).Up(ctx); err != nil {
		t.Fatalf("apply backtest identity migration: %v", err)
	}
	var runKey string
	var snapshot sql.NullString
	var status string
	var runID string
	if err := db.QueryRowContext(ctx, "SELECT run_id,run_key,snapshot_hash,status FROM t_backtest_run WHERE run_id=?", legacyID).Scan(&runID, &runKey, &snapshot, &status); err != nil {
		t.Fatalf("read migrated legacy run: %v", err)
	}
	var wantKey string
	if err := db.QueryRowContext(ctx, "SELECT SHA2(CONCAT('legacy:',?),256)", legacyID).Scan(&wantKey); err != nil {
		t.Fatal(err)
	}
	if runID != legacyID || runKey != wantKey || snapshot.Valid || status != "SUCCESS" {
		t.Fatalf("legacy row after migration = (%s,%s,%v,%s), want preserved identity/status and null unknown snapshot", runID, runKey, snapshot, status)
	}
	repository, err := NewBacktestRunRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	legacyRun, err := repository.Find(ctx, legacyID)
	if err != nil || legacyRun.RunKey != wantKey || legacyRun.SnapshotHash != "" || legacyRun.Status != "SUCCESS" {
		t.Fatalf("repository read of legacy run = %#v, %v", legacyRun, err)
	}
}

func TestMigrateConcurrentUpSerializesMySQL(t *testing.T) {
	db := openIsolatedMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	migrator := newEmbeddedMigrator(t, db)

	start := make(chan struct{})
	errors := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			errors <- migrator.Up(ctx)
		}()
	}
	close(start)
	for range 2 {
		if err := <-errors; err != nil {
			t.Fatalf("concurrent up: %v", err)
		}
	}
	assertMigrationState(t, ctx, db, 1, false, "up")
}

func TestMigrateUnavailableDatabaseReturnsCause(t *testing.T) {
	db, err := sql.Open("mysql", "user:password@unix(/tmp/stock-quant-missing-mysql.sock)/stock_quant")
	if err != nil {
		t.Fatalf("open lazy database handle: %v", err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	migrator := &Migrator{DB: db, Migrations: []Migration{{Version: 1, Name: "empty", Up: "SELECT 1;", Down: "SELECT 1;"}}}
	if err := migrator.Up(ctx); err == nil || !strings.Contains(err.Error(), "acquire migration lock") {
		t.Fatalf("up error = %v, want wrapped lock/connection failure", err)
	}
}

func openIsolatedMySQL(t *testing.T) *sql.DB {
	t.Helper()
	return openIsolatedMySQLWithOptions(t, false)
}

func openIsolatedMySQLClientFoundRows(t *testing.T) *sql.DB {
	t.Helper()
	return openIsolatedMySQLWithOptions(t, true)
}

func openIsolatedMySQLWithOptions(t *testing.T, clientFoundRows bool) *sql.DB {
	t.Helper()
	dsn := os.Getenv("MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("MYSQL_TEST_DSN is not set; real MySQL integration test skipped")
	}
	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("parse MYSQL_TEST_DSN: %v", err)
	}
	cfg.ClientFoundRows = clientFoundRows
	cfg.DBName = ""
	admin, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatalf("open MySQL admin connection: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := admin.PingContext(ctx); err != nil {
		admin.Close()
		t.Fatalf("connect to MySQL test service: %v", err)
	}
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		admin.Close()
		t.Fatalf("generate isolated database name: %v", err)
	}
	databaseName := "sq_migration_test_" + hex.EncodeToString(random[:])
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE `"+databaseName+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"); err != nil {
		admin.Close()
		t.Fatalf("create isolated database: %v", err)
	}
	cfg.DBName = databaseName
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		_, _ = admin.ExecContext(ctx, "DROP DATABASE `"+databaseName+"`")
		admin.Close()
		t.Fatalf("open isolated database: %v", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		_, _ = admin.ExecContext(ctx, "DROP DATABASE `"+databaseName+"`")
		admin.Close()
		t.Fatalf("connect to isolated database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if _, err := admin.ExecContext(cleanupCtx, "DROP DATABASE `"+databaseName+"`"); err != nil {
			t.Errorf("drop isolated test database %s: %v", databaseName, err)
		}
		_ = admin.Close()
	})
	return db
}

func newEmbeddedMigrator(t *testing.T, db *sql.DB) *Migrator {
	t.Helper()
	items, err := LoadMigrations(migrations.FS)
	if err != nil {
		t.Fatalf("load embedded migrations: %v", err)
	}
	return &Migrator{DB: db, Migrations: items}
}

func countTables(t *testing.T, ctx context.Context, db *sql.DB) int {
	t.Helper()
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE()").Scan(&count); err != nil {
		t.Fatalf("count tables: %v", err)
	}
	return count
}

func tableExists(t *testing.T, ctx context.Context, db *sql.DB, name string) bool {
	t.Helper()
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", name).Scan(&count); err != nil {
		t.Fatalf("check table %s: %v", name, err)
	}
	return count == 1
}

func columnExists(t *testing.T, ctx context.Context, db *sql.DB, table, column string) bool {
	t.Helper()
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?", table, column).Scan(&count); err != nil {
		t.Fatalf("check column %s.%s: %v", table, column, err)
	}
	return count == 1
}

func assertColumnExists(t *testing.T, ctx context.Context, db *sql.DB, table, column string) {
	t.Helper()
	if !columnExists(t, ctx, db, table, column) {
		t.Errorf("expected column %s.%s to exist", table, column)
	}
}

func assertTablesExist(t *testing.T, ctx context.Context, db *sql.DB, names []string) {
	t.Helper()
	for _, name := range names {
		if !tableExists(t, ctx, db, name) {
			t.Errorf("expected table %s to exist", name)
		}
	}
}

func assertIndexExists(t *testing.T, ctx context.Context, db *sql.DB, table, index string) {
	t.Helper()
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?", table, index).Scan(&count)
	if err != nil {
		t.Fatalf("check index %s on %s: %v", index, table, err)
	}
	if count == 0 {
		t.Errorf("expected index %s on %s", index, table)
	}
}

func assertMigrationState(t *testing.T, ctx context.Context, db *sql.DB, version int, dirty bool, direction string) {
	t.Helper()
	var gotVersion int
	var gotDirty bool
	var gotDirection string
	err := db.QueryRowContext(ctx, "SELECT version, dirty, direction FROM t_schema_migrations WHERE version = ?", version).Scan(&gotVersion, &gotDirty, &gotDirection)
	if err != nil {
		t.Fatalf("read migration state: %v", err)
	}
	if gotVersion != version || gotDirty != dirty || gotDirection != direction {
		t.Fatalf("migration state = (%d, %t, %q), want (%d, %t, %q)", gotVersion, gotDirty, gotDirection, version, dirty, direction)
	}
}

func TestLoadMigrationsRequiresPairedContiguousVersions(t *testing.T) {
	valid := fstest.MapFS{
		"0001.up.sql":   &fstest.MapFile{Data: []byte("CREATE TABLE t_one (id INT);")},
		"0001.down.sql": &fstest.MapFile{Data: []byte("DROP TABLE t_one;")},
	}
	loaded, err := LoadMigrations(valid)
	if err != nil || len(loaded) != 1 || loaded[0].Version != 1 {
		t.Fatalf("LoadMigrations() = %#v, %v; want version 1", loaded, err)
	}

	orphanDown := fstest.MapFS{
		"0001.up.sql":   &fstest.MapFile{Data: []byte("CREATE TABLE t_one (id INT);")},
		"0001.down.sql": &fstest.MapFile{Data: []byte("DROP TABLE t_one;")},
		"0002.down.sql": &fstest.MapFile{Data: []byte("DROP TABLE t_two;")},
	}
	if _, err := LoadMigrations(orphanDown); err == nil || !strings.Contains(err.Error(), "no matching up") {
		t.Fatalf("LoadMigrations() error = %v, want orphan down error", err)
	}
}

func TestSplitMigrationSQLRespectsQuotedSemicolonsAndComments(t *testing.T) {
	source := "SELECT 'a;b'; -- this ; stays in a comment\nSELECT `column;name` FROM t_test; /* and ; here */ SELECT 3;"
	statements, err := splitStatements(source)
	if err != nil {
		t.Fatalf("splitStatements() error = %v", err)
	}
	if len(statements) != 3 {
		t.Fatalf("splitStatements() returned %d statements, want 3: %#v", len(statements), statements)
	}
	if statements[0] != "SELECT 'a;b'" || statements[1] != "SELECT `column;name` FROM t_test" || statements[2] != "SELECT 3" {
		t.Fatalf("splitStatements() = %#v", statements)
	}
}

func TestMigrateRejectsChangedAppliedChecksumMySQL(t *testing.T) {
	db := openIsolatedMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	migrator := newEmbeddedMigrator(t, db)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("apply original migration: %v", err)
	}

	changed := append([]Migration(nil), migrator.Migrations...)
	changed[0].Up += "\n-- changed after application\n"
	if err := (&Migrator{DB: db, Migrations: changed}).Up(ctx); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("up with changed migration = %v, want checksum mismatch", err)
	}
	assertMigrationState(t, ctx, db, 1, false, "up")
}

func TestMigrateRefusesToAdoptPreexistingTableMySQL(t *testing.T) {
	db := openIsolatedMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := db.ExecContext(ctx, "CREATE TABLE t_stock (fixture_id INT PRIMARY KEY)"); err != nil {
		t.Fatalf("create preexisting table: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO t_stock (fixture_id) VALUES (42)"); err != nil {
		t.Fatalf("insert preexisting row: %v", err)
	}

	migrator := newEmbeddedMigrator(t, db)
	if err := migrator.Up(ctx); err == nil || !strings.Contains(err.Error(), "pre-existing table") {
		t.Fatalf("up with a preexisting migration object = %v, want adoption refusal", err)
	}
	var fixtureID int
	if err := db.QueryRowContext(ctx, "SELECT fixture_id FROM t_stock").Scan(&fixtureID); err != nil || fixtureID != 42 {
		t.Fatalf("preexisting table data = %d, %v; want 42 and no error", fixtureID, err)
	}
	var applied int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM t_schema_migrations WHERE version = 1").Scan(&applied); err != nil {
		t.Fatalf("read migration ledger: %v", err)
	}
	if applied != 0 {
		t.Fatalf("migration ledger contains %d applied version rows, want 0", applied)
	}
}
