-- Add immutable backtest identity while retaining and identifying legacy rows.
SET @has_run_key = (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='t_backtest_run' AND column_name='run_key');
SET @ddl = IF(@has_run_key=0, 'ALTER TABLE t_backtest_run ADD COLUMN run_key CHAR(64) NULL AFTER run_id', 'SELECT 1');
PREPARE migration_stmt FROM @ddl;
EXECUTE migration_stmt;
DEALLOCATE PREPARE migration_stmt;

SET @has_snapshot_hash = (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='t_backtest_run' AND column_name='snapshot_hash');
SET @ddl = IF(@has_snapshot_hash=0, 'ALTER TABLE t_backtest_run ADD COLUMN snapshot_hash CHAR(64) NULL AFTER config_hash', 'SELECT 1');
PREPARE migration_stmt FROM @ddl;
EXECUTE migration_stmt;
DEALLOCATE PREPARE migration_stmt;

SET @has_error_message = (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='t_backtest_run' AND column_name='error_message');
SET @ddl = IF(@has_error_message=0, 'ALTER TABLE t_backtest_run ADD COLUMN error_message VARCHAR(1000) NULL AFTER risk_json', 'SELECT 1');
PREPARE migration_stmt FROM @ddl;
EXECUTE migration_stmt;
DEALLOCATE PREPARE migration_stmt;

UPDATE t_backtest_run SET run_key=SHA2(CONCAT('legacy:',run_id),256) WHERE run_key IS NULL OR run_key='';
ALTER TABLE t_backtest_run MODIFY COLUMN run_key CHAR(64) NOT NULL;

SET @has_run_key_index = (SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name='t_backtest_run' AND index_name='uq_backtest_run_key');
SET @ddl = IF(@has_run_key_index=0, 'ALTER TABLE t_backtest_run ADD UNIQUE KEY uq_backtest_run_key(run_key)', 'SELECT 1');
PREPARE migration_stmt FROM @ddl;
EXECUTE migration_stmt;
DEALLOCATE PREPARE migration_stmt;
