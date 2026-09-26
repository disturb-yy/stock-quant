CREATE TABLE IF NOT EXISTS t_instrument (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    symbol VARCHAR(32) NOT NULL,
    market VARCHAR(16) NOT NULL,
    name VARCHAR(128) NOT NULL,
    status VARCHAR(32) NOT NULL,
    source_provider VARCHAR(32) NOT NULL,
    source_mode VARCHAR(32) NOT NULL,
    data_as_of DATE NULL,
    updated_at DATETIME(6) NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT uk_t_instrument_market_symbol UNIQUE (market, symbol),
    KEY idx_t_instrument_updated_at (updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS t_instrument_daily_bar (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    symbol VARCHAR(32) NOT NULL,
    market VARCHAR(16) NOT NULL,
    trade_date DATE NOT NULL,
    open_price DECIMAL(20, 6) NOT NULL,
    high_price DECIMAL(20, 6) NOT NULL,
    low_price DECIMAL(20, 6) NOT NULL,
    close_price DECIMAL(20, 6) NOT NULL,
    volume DECIMAL(24, 6) NOT NULL,
    source_provider VARCHAR(32) NOT NULL,
    source_mode VARCHAR(32) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT uk_t_instrument_daily_bar_market_symbol_date
        UNIQUE (market, symbol, trade_date),
    KEY idx_t_instrument_daily_bar_symbol_date (symbol, trade_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS t_data_sync_task (
    id CHAR(36) NOT NULL,
    target VARCHAR(32) NOT NULL,
    trigger_type VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL,
    active_target VARCHAR(32) NULL,
    source_provider VARCHAR(32) NOT NULL,
    source_mode VARCHAR(32) NOT NULL,
    start_date DATE NULL,
    end_date DATE NULL,
    data_as_of DATE NULL,
    created_at DATETIME(6) NOT NULL,
    started_at DATETIME(6) NULL,
    finished_at DATETIME(6) NULL,
    updated_at DATETIME(6) NOT NULL,
    next_attempt_at DATETIME(6) NULL,
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL,
    failure_reason TEXT NULL,
    processed_count INT NOT NULL DEFAULT 0,
    created_count INT NOT NULL DEFAULT 0,
    updated_count INT NOT NULL DEFAULT 0,
    failed_count INT NOT NULL DEFAULT 0,
    retry_of_task_id CHAR(36) NULL,
    PRIMARY KEY (id),
    CONSTRAINT uk_t_data_sync_task_active_target UNIQUE (active_target),
    KEY idx_t_data_sync_task_queue (status, next_attempt_at, created_at),
    KEY idx_t_data_sync_task_updated_at (updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
