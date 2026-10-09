-- Initial schema migration for MySQL 8.4.
-- Dates are market trading dates; timestamps use UTC. Never put production secrets here.
CREATE TABLE IF NOT EXISTS t_stock (
  ts_code VARCHAR(16) PRIMARY KEY, name VARCHAR(128) NOT NULL,
  exchange VARCHAR(8) NOT NULL, market VARCHAR(32) NOT NULL,
  list_date DATE NOT NULL, delist_date DATE NULL,
  list_status VARCHAR(8) NOT NULL, updated_at DATETIME(6) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
CREATE TABLE IF NOT EXISTS t_trade_calendar (
  exchange VARCHAR(8) NOT NULL, cal_date DATE NOT NULL,
  is_open BOOLEAN NOT NULL, pretrade_date DATE NULL,
  PRIMARY KEY(exchange,cal_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
CREATE TABLE IF NOT EXISTS t_daily_price (
  ts_code VARCHAR(16) NOT NULL, trade_date DATE NOT NULL,
  open DECIMAL(20,6) NOT NULL, high DECIMAL(20,6) NOT NULL,
  low DECIMAL(20,6) NOT NULL, close DECIMAL(20,6) NOT NULL,
  amount_yuan DECIMAL(24,4) NOT NULL, volume_lot DECIMAL(24,4) NOT NULL,
  source_hash CHAR(64) NOT NULL, revision INT NOT NULL DEFAULT 1,
  fetched_at DATETIME(6) NOT NULL,
  PRIMARY KEY(ts_code,trade_date), KEY idx_daily_by_date(trade_date,ts_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
CREATE TABLE IF NOT EXISTS t_adj_factor (
  ts_code VARCHAR(16) NOT NULL, trade_date DATE NOT NULL,
  adj_factor DECIMAL(24,10) NOT NULL, source_hash CHAR(64) NOT NULL,
  revision INT NOT NULL DEFAULT 1, fetched_at DATETIME(6) NOT NULL,
  PRIMARY KEY(ts_code,trade_date), KEY idx_adj_by_date(trade_date,ts_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
CREATE TABLE IF NOT EXISTS t_stock_status_daily (
  ts_code VARCHAR(16) NOT NULL, trade_date DATE NOT NULL,
  st_status ENUM('ST','NON_ST','UNKNOWN') NOT NULL DEFAULT 'UNKNOWN',
  trade_status ENUM('NORMAL','SUSPENDED','UNKNOWN') NOT NULL DEFAULT 'UNKNOWN',
  provenance VARCHAR(64) NOT NULL, confidence VARCHAR(16) NOT NULL,
  PRIMARY KEY(ts_code,trade_date), KEY idx_status_date(trade_date,ts_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
CREATE TABLE IF NOT EXISTS t_sync_job (
  job_id CHAR(26) PRIMARY KEY, task_key VARCHAR(128) NOT NULL UNIQUE,
  api_name VARCHAR(64) NOT NULL, trade_date DATE NULL,
  status VARCHAR(16) NOT NULL, attempts INT NOT NULL DEFAULT 0,
  requested_at DATETIME(6) NOT NULL, expected_rows INT NULL, received_rows INT NULL,
  quality_report_json JSON NULL, last_error VARCHAR(1000) NULL,
  created_at DATETIME(6) NOT NULL, updated_at DATETIME(6) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
CREATE TABLE IF NOT EXISTS t_data_snapshot (
  snapshot_id CHAR(26) PRIMARY KEY, as_of DATE NOT NULL,
  snapshot_hash CHAR(64) NOT NULL UNIQUE, data_revision VARCHAR(100) NOT NULL,
  data_ref VARCHAR(512) NULL, is_complete BOOLEAN NOT NULL,
  st_quality VARCHAR(32) NOT NULL, report_json JSON NOT NULL,
  created_at DATETIME(6) NOT NULL, KEY idx_snapshot_date(as_of,created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
CREATE TABLE IF NOT EXISTS t_strategy (
  strategy_id VARCHAR(80) NOT NULL, strategy_version VARCHAR(32) NOT NULL,
  config_hash CHAR(64) NOT NULL, config_json JSON NOT NULL,
  created_at DATETIME(6) NOT NULL,
  PRIMARY KEY(strategy_id,strategy_version)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
CREATE TABLE IF NOT EXISTS t_screening_run (
  run_id CHAR(26) PRIMARY KEY, run_key CHAR(64) NOT NULL UNIQUE,
  strategy_id VARCHAR(80) NOT NULL, strategy_version VARCHAR(32) NOT NULL,
  as_of DATE NOT NULL, config_hash CHAR(64) NOT NULL,
  snapshot_hash CHAR(64) NOT NULL, status VARCHAR(16) NOT NULL,
  total_universe INT NOT NULL DEFAULT 0, total_eligible INT NOT NULL DEFAULT 0,
  error_code VARCHAR(80) NULL, error_message VARCHAR(1000) NULL,
  rejection_summary_json JSON NULL,
  created_at DATETIME(6) NOT NULL, updated_at DATETIME(6) NOT NULL,
  KEY idx_screen_strategy_date(strategy_id,as_of)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
CREATE TABLE IF NOT EXISTS t_screening_result (
  run_id CHAR(26) NOT NULL, ts_code VARCHAR(16) NOT NULL,
  final_rank INT NULL, total_score DOUBLE NULL,
  raw_factors_json JSON NULL, factor_scores_json JSON NULL,
  reason_json JSON NULL,
  PRIMARY KEY(run_id,ts_code), KEY idx_screen_rank(run_id,final_rank)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
CREATE TABLE IF NOT EXISTS t_backtest_run (
  run_id CHAR(26) PRIMARY KEY, strategy_id VARCHAR(80) NOT NULL,
  strategy_version VARCHAR(32) NOT NULL, start_date DATE NOT NULL,
  end_date DATE NOT NULL, config_hash CHAR(64) NOT NULL,
  mode VARCHAR(32) NOT NULL, status VARCHAR(16) NOT NULL,
  metrics_json JSON NULL, risk_json JSON NULL,
  created_at DATETIME(6) NOT NULL, updated_at DATETIME(6) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
CREATE TABLE IF NOT EXISTS t_backtest_equity (
  run_id CHAR(26) NOT NULL, trade_date DATE NOT NULL,
  equity DECIMAL(24,4) NOT NULL, cash DECIMAL(24,4) NOT NULL,
  exposure DECIMAL(24,4) NOT NULL, benchmark_equity DECIMAL(24,4) NULL,
  PRIMARY KEY(run_id,trade_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
CREATE TABLE IF NOT EXISTS t_backtest_trade (
  id BIGINT AUTO_INCREMENT PRIMARY KEY, run_id CHAR(26) NOT NULL,
  trade_date DATE NOT NULL, ts_code VARCHAR(16) NOT NULL,
  side VARCHAR(4) NOT NULL, intended_qty BIGINT NOT NULL,
  filled_qty BIGINT NOT NULL, filled_price DECIMAL(20,6) NULL,
  fee_yuan DECIMAL(24,4) NOT NULL DEFAULT 0,
  rejection_reason VARCHAR(100) NULL,
  KEY idx_backtest_trade_run(run_id,trade_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
