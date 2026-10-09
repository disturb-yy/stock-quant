-- Remove only the columns and index introduced by 0002; preserve every run row.
SET @has_run_key_index = (SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name='t_backtest_run' AND index_name='uq_backtest_run_key');
SET @ddl = IF(@has_run_key_index>0, 'ALTER TABLE t_backtest_run DROP INDEX uq_backtest_run_key', 'SELECT 1');
PREPARE migration_stmt FROM @ddl;
EXECUTE migration_stmt;
DEALLOCATE PREPARE migration_stmt;

SET @has_error_message = (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='t_backtest_run' AND column_name='error_message');
SET @ddl = IF(@has_error_message>0, 'ALTER TABLE t_backtest_run DROP COLUMN error_message', 'SELECT 1');
PREPARE migration_stmt FROM @ddl;
EXECUTE migration_stmt;
DEALLOCATE PREPARE migration_stmt;

SET @has_snapshot_hash = (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='t_backtest_run' AND column_name='snapshot_hash');
SET @ddl = IF(@has_snapshot_hash>0, 'ALTER TABLE t_backtest_run DROP COLUMN snapshot_hash', 'SELECT 1');
PREPARE migration_stmt FROM @ddl;
EXECUTE migration_stmt;
DEALLOCATE PREPARE migration_stmt;

SET @has_run_key = (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='t_backtest_run' AND column_name='run_key');
SET @ddl = IF(@has_run_key>0, 'ALTER TABLE t_backtest_run DROP COLUMN run_key', 'SELECT 1');
PREPARE migration_stmt FROM @ddl;
EXECUTE migration_stmt;
DEALLOCATE PREPARE migration_stmt;
