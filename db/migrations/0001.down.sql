-- Reverse only objects created by 0001.up.sql.
DROP TABLE IF EXISTS t_backtest_trade;
DROP TABLE IF EXISTS t_backtest_equity;
DROP TABLE IF EXISTS t_backtest_run;
DROP TABLE IF EXISTS t_screening_result;
DROP TABLE IF EXISTS t_screening_run;
DROP TABLE IF EXISTS t_strategy;
DROP TABLE IF EXISTS t_data_snapshot;
DROP TABLE IF EXISTS t_sync_job;
DROP TABLE IF EXISTS t_stock_status_daily;
DROP TABLE IF EXISTS t_adj_factor;
DROP TABLE IF EXISTS t_daily_price;
DROP TABLE IF EXISTS t_trade_calendar;
DROP TABLE IF EXISTS t_stock;
