package mysql

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"stock-quant/internal/market/domain"
	"stock-quant/internal/market/ports"
	"stock-quant/internal/shared/types"
)

type stockRepository struct{ db *sql.DB }
type tradeCalendarRepository struct{ db *sql.DB }
type dailyPriceRepository struct{ db *sql.DB }
type adjFactorRepository struct{ db *sql.DB }

func NewStockRepository(db *sql.DB) (ports.StockRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("create stock repository: database is required")
	}
	return &stockRepository{db: db}, nil
}

func NewTradeCalendarRepository(db *sql.DB) (ports.TradeCalendarRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("create trade calendar repository: database is required")
	}
	return &tradeCalendarRepository{db: db}, nil
}

func NewDailyPriceRepository(db *sql.DB) (ports.DailyPriceRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("create daily price repository: database is required")
	}
	return &dailyPriceRepository{db: db}, nil
}

func NewAdjFactorRepository(db *sql.DB) (ports.AdjFactorRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("create adjustment factor repository: database is required")
	}
	return &adjFactorRepository{db: db}, nil
}

var _ ports.StockRepository = (*stockRepository)(nil)
var _ ports.TradeCalendarRepository = (*tradeCalendarRepository)(nil)
var _ ports.DailyPriceRepository = (*dailyPriceRepository)(nil)
var _ ports.AdjFactorRepository = (*adjFactorRepository)(nil)

func (r *stockRepository) Upsert(ctx context.Context, stocks []domain.Stock) error {
	return inTransaction(ctx, r.db, "upsert stocks", func(tx *sql.Tx) error {
		const query = `INSERT INTO t_stock (ts_code,name,exchange,market,list_date,delist_date,list_status,updated_at)
VALUES (?,?,?,?,?,?,?,?) AS incoming ON DUPLICATE KEY UPDATE name=incoming.name,exchange=incoming.exchange,market=incoming.market,list_date=incoming.list_date,delist_date=incoming.delist_date,list_status=incoming.list_status,updated_at=incoming.updated_at`
		for i, stock := range stocks {
			if err := validateStock(stock); err != nil {
				return fmt.Errorf("stock row %d: %w", i, err)
			}
			delistDate, err := nullableDate(stock.DelistDate)
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, query, stock.TSCode, stock.Name, stock.Exchange, stock.Market, stock.ListDate.String(), delistDate, stock.ListStatus, databaseTimestamp(stock.UpdatedAt)); err != nil {
				return fmt.Errorf("upsert stock %s: %w", stock.TSCode, err)
			}
		}
		return nil
	})
}

func (r *stockRepository) Find(ctx context.Context, tsCode string) (domain.Stock, error) {
	var stock domain.Stock
	var listDate string
	var delistDate sql.NullString
	var updatedAt string
	err := r.db.QueryRowContext(ctx, `SELECT ts_code,name,exchange,market,CAST(list_date AS CHAR),CAST(delist_date AS CHAR),list_status,DATE_FORMAT(updated_at,'%Y-%m-%d %H:%i:%s.%f') FROM t_stock WHERE ts_code=?`, tsCode).Scan(&stock.TSCode, &stock.Name, &stock.Exchange, &stock.Market, &listDate, &delistDate, &stock.ListStatus, &updatedAt)
	if err != nil {
		return domain.Stock{}, fmt.Errorf("find stock %s: %w", tsCode, err)
	}
	stock.ListDate, err = types.ParseTradingDate(listDate)
	if err != nil {
		return domain.Stock{}, fmt.Errorf("parse stock list date: %w", err)
	}
	if delistDate.Valid {
		date, parseErr := types.ParseTradingDate(delistDate.String)
		if parseErr != nil {
			return domain.Stock{}, fmt.Errorf("parse stock delist date: %w", parseErr)
		}
		stock.DelistDate = &date
	}
	stock.UpdatedAt, err = parseDatabaseTimestamp(updatedAt)
	if err != nil {
		return domain.Stock{}, fmt.Errorf("parse stock update time: %w", err)
	}
	return stock, nil
}

func (r *tradeCalendarRepository) Upsert(ctx context.Context, dates []domain.TradeCalendar) error {
	return inTransaction(ctx, r.db, "upsert trade calendar", func(tx *sql.Tx) error {
		const query = `INSERT INTO t_trade_calendar (exchange,cal_date,is_open,pretrade_date) VALUES (?,?,?,?) AS incoming ON DUPLICATE KEY UPDATE is_open=incoming.is_open,pretrade_date=incoming.pretrade_date`
		for i, item := range dates {
			if err := validateCodeDate(item.Exchange, item.Date, "exchange"); err != nil {
				return fmt.Errorf("calendar row %d: %w", i, err)
			}
			pretradeDate, err := nullableDate(item.PretradeDate)
			if err != nil {
				return fmt.Errorf("calendar row %d: %w", i, err)
			}
			if _, err := tx.ExecContext(ctx, query, item.Exchange, item.Date.String(), item.IsOpen, pretradeDate); err != nil {
				return fmt.Errorf("upsert calendar %s/%s: %w", item.Exchange, item.Date, err)
			}
		}
		return nil
	})
}

func (r *tradeCalendarRepository) List(ctx context.Context, exchange string, from, through types.TradingDate) ([]domain.TradeCalendar, error) {
	if err := validateRange(exchange, from, through, "exchange"); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT exchange,CAST(cal_date AS CHAR),is_open,CAST(pretrade_date AS CHAR) FROM t_trade_calendar WHERE exchange=? AND cal_date BETWEEN ? AND ? ORDER BY cal_date ASC`, exchange, from.String(), through.String())
	if err != nil {
		return nil, fmt.Errorf("list trade calendar: %w", err)
	}
	defer rows.Close()
	result := make([]domain.TradeCalendar, 0)
	for rows.Next() {
		var item domain.TradeCalendar
		var date string
		var pretrade sql.NullString
		if err := rows.Scan(&item.Exchange, &date, &item.IsOpen, &pretrade); err != nil {
			return nil, fmt.Errorf("scan trade calendar: %w", err)
		}
		item.Date, err = types.ParseTradingDate(date)
		if err != nil {
			return nil, fmt.Errorf("parse calendar date: %w", err)
		}
		if pretrade.Valid {
			parsed, parseErr := types.ParseTradingDate(pretrade.String)
			if parseErr != nil {
				return nil, fmt.Errorf("parse pretrade date: %w", parseErr)
			}
			item.PretradeDate = &parsed
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read trade calendar: %w", err)
	}
	return result, nil
}

func (r *dailyPriceRepository) Upsert(ctx context.Context, prices []domain.DailyPrice) error {
	return inTransaction(ctx, r.db, "upsert daily prices", func(tx *sql.Tx) error {
		const query = `INSERT INTO t_daily_price (ts_code,trade_date,open,high,low,close,amount_yuan,volume_lot,source_hash,revision,fetched_at)
VALUES (?,?,?,?,?,?,?,?,?,1,?) AS incoming ON DUPLICATE KEY UPDATE
revision=t_daily_price.revision+IF(t_daily_price.open<>incoming.open OR t_daily_price.high<>incoming.high OR t_daily_price.low<>incoming.low OR t_daily_price.close<>incoming.close OR t_daily_price.amount_yuan<>incoming.amount_yuan OR t_daily_price.volume_lot<>incoming.volume_lot OR t_daily_price.source_hash<>incoming.source_hash,1,0),
open=incoming.open,high=incoming.high,low=incoming.low,close=incoming.close,amount_yuan=incoming.amount_yuan,volume_lot=incoming.volume_lot,source_hash=incoming.source_hash,fetched_at=incoming.fetched_at`
		for i, item := range prices {
			if err := validateDailyPrice(item); err != nil {
				return fmt.Errorf("daily price row %d: %w", i, err)
			}
			if _, err := tx.ExecContext(ctx, query, item.TSCode, item.TradeDate.String(), item.Open, item.High, item.Low, item.Close, item.AmountYuan.Float64(), item.VolumeLot, item.SourceHash, databaseTimestamp(item.FetchedAt)); err != nil {
				return fmt.Errorf("upsert daily price %s/%s: %w", item.TSCode, item.TradeDate, err)
			}
		}
		return nil
	})
}

func (r *dailyPriceRepository) ListByCode(ctx context.Context, tsCode string, from, through types.TradingDate) ([]domain.DailyPrice, error) {
	if err := validateRange(tsCode, from, through, "stock code"); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT ts_code,CAST(trade_date AS CHAR),open,high,low,close,amount_yuan,volume_lot,source_hash,revision,DATE_FORMAT(fetched_at,'%Y-%m-%d %H:%i:%s.%f') FROM t_daily_price WHERE ts_code=? AND trade_date BETWEEN ? AND ? ORDER BY trade_date ASC`, tsCode, from.String(), through.String())
	if err != nil {
		return nil, fmt.Errorf("list daily prices: %w", err)
	}
	defer rows.Close()
	result := make([]domain.DailyPrice, 0)
	for rows.Next() {
		var item domain.DailyPrice
		var date, fetchedAt string
		var amount float64
		if err := rows.Scan(&item.TSCode, &date, &item.Open, &item.High, &item.Low, &item.Close, &amount, &item.VolumeLot, &item.SourceHash, &item.Revision, &fetchedAt); err != nil {
			return nil, fmt.Errorf("scan daily price: %w", err)
		}
		item.TradeDate, err = types.ParseTradingDate(date)
		if err != nil {
			return nil, fmt.Errorf("parse daily trade date: %w", err)
		}
		item.AmountYuan, err = types.NewAmountYuan(amount)
		if err != nil {
			return nil, fmt.Errorf("parse daily amount: %w", err)
		}
		item.FetchedAt, err = parseDatabaseTimestamp(fetchedAt)
		if err != nil {
			return nil, fmt.Errorf("parse daily fetched time: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read daily prices: %w", err)
	}
	return result, nil
}

func (r *dailyPriceRepository) ListByDate(ctx context.Context, date types.TradingDate) ([]domain.DailyPrice, error) {
	if !date.Valid() {
		return nil, fmt.Errorf("list daily prices by date: valid trading date is required")
	}
	rows, err := r.db.QueryContext(ctx, `SELECT ts_code,CAST(trade_date AS CHAR),open,high,low,close,amount_yuan,volume_lot,source_hash,revision,DATE_FORMAT(fetched_at,'%Y-%m-%d %H:%i:%s.%f') FROM t_daily_price WHERE trade_date=? ORDER BY ts_code ASC`, date.String())
	if err != nil {
		return nil, fmt.Errorf("list daily prices by date: %w", err)
	}
	defer rows.Close()
	result := make([]domain.DailyPrice, 0)
	for rows.Next() {
		var item domain.DailyPrice
		var tradeDate, fetchedAt string
		var amount float64
		if err := rows.Scan(&item.TSCode, &tradeDate, &item.Open, &item.High, &item.Low, &item.Close, &amount, &item.VolumeLot, &item.SourceHash, &item.Revision, &fetchedAt); err != nil {
			return nil, fmt.Errorf("scan daily price by date: %w", err)
		}
		item.TradeDate, err = types.ParseTradingDate(tradeDate)
		if err != nil {
			return nil, fmt.Errorf("parse daily trade date: %w", err)
		}
		item.AmountYuan, err = types.NewAmountYuan(amount)
		if err != nil {
			return nil, fmt.Errorf("parse daily amount: %w", err)
		}
		item.FetchedAt, err = parseDatabaseTimestamp(fetchedAt)
		if err != nil {
			return nil, fmt.Errorf("parse daily fetched time: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read daily prices by date: %w", err)
	}
	return result, nil
}

func (r *adjFactorRepository) Upsert(ctx context.Context, factors []domain.AdjFactor) error {
	return inTransaction(ctx, r.db, "upsert adjustment factors", func(tx *sql.Tx) error {
		const query = `INSERT INTO t_adj_factor (ts_code,trade_date,adj_factor,source_hash,revision,fetched_at) VALUES (?,?,?,?,1,?) AS incoming ON DUPLICATE KEY UPDATE
revision=t_adj_factor.revision+IF(t_adj_factor.adj_factor<>incoming.adj_factor OR t_adj_factor.source_hash<>incoming.source_hash,1,0),adj_factor=incoming.adj_factor,source_hash=incoming.source_hash,fetched_at=incoming.fetched_at`
		for i, item := range factors {
			if err := validateAdjFactor(item); err != nil {
				return fmt.Errorf("adjustment factor row %d: %w", i, err)
			}
			if _, err := tx.ExecContext(ctx, query, item.TSCode, item.TradeDate.String(), item.Factor, item.SourceHash, databaseTimestamp(item.FetchedAt)); err != nil {
				return fmt.Errorf("upsert adjustment factor %s/%s: %w", item.TSCode, item.TradeDate, err)
			}
		}
		return nil
	})
}

func (r *adjFactorRepository) ListByCode(ctx context.Context, tsCode string, from, through types.TradingDate) ([]domain.AdjFactor, error) {
	if err := validateRange(tsCode, from, through, "stock code"); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT ts_code,CAST(trade_date AS CHAR),adj_factor,source_hash,revision,DATE_FORMAT(fetched_at,'%Y-%m-%d %H:%i:%s.%f') FROM t_adj_factor WHERE ts_code=? AND trade_date BETWEEN ? AND ? ORDER BY trade_date ASC`, tsCode, from.String(), through.String())
	if err != nil {
		return nil, fmt.Errorf("list adjustment factors: %w", err)
	}
	defer rows.Close()
	result := make([]domain.AdjFactor, 0)
	for rows.Next() {
		var item domain.AdjFactor
		var date, fetchedAt string
		if err := rows.Scan(&item.TSCode, &date, &item.Factor, &item.SourceHash, &item.Revision, &fetchedAt); err != nil {
			return nil, fmt.Errorf("scan adjustment factor: %w", err)
		}
		item.TradeDate, err = types.ParseTradingDate(date)
		if err != nil {
			return nil, fmt.Errorf("parse adjustment factor date: %w", err)
		}
		item.FetchedAt, err = parseDatabaseTimestamp(fetchedAt)
		if err != nil {
			return nil, fmt.Errorf("parse adjustment factor fetched time: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read adjustment factors: %w", err)
	}
	return result, nil
}

func (r *adjFactorRepository) ListByDate(ctx context.Context, date types.TradingDate) ([]domain.AdjFactor, error) {
	if !date.Valid() {
		return nil, fmt.Errorf("list adjustment factors by date: valid trading date is required")
	}
	rows, err := r.db.QueryContext(ctx, `SELECT ts_code,CAST(trade_date AS CHAR),adj_factor,source_hash,revision,DATE_FORMAT(fetched_at,'%Y-%m-%d %H:%i:%s.%f') FROM t_adj_factor WHERE trade_date=? ORDER BY ts_code ASC`, date.String())
	if err != nil {
		return nil, fmt.Errorf("list adjustment factors by date: %w", err)
	}
	defer rows.Close()
	result := make([]domain.AdjFactor, 0)
	for rows.Next() {
		var item domain.AdjFactor
		var tradeDate, fetchedAt string
		if err := rows.Scan(&item.TSCode, &tradeDate, &item.Factor, &item.SourceHash, &item.Revision, &fetchedAt); err != nil {
			return nil, fmt.Errorf("scan adjustment factor by date: %w", err)
		}
		item.TradeDate, err = types.ParseTradingDate(tradeDate)
		if err != nil {
			return nil, fmt.Errorf("parse adjustment factor date: %w", err)
		}
		item.FetchedAt, err = parseDatabaseTimestamp(fetchedAt)
		if err != nil {
			return nil, fmt.Errorf("parse adjustment factor fetched time: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read adjustment factors by date: %w", err)
	}
	return result, nil
}

func inTransaction(ctx context.Context, db *sql.DB, operation string, apply func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: begin transaction: %w", operation, err)
	}
	defer tx.Rollback()
	if err := apply(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: commit: %w", operation, err)
	}
	return nil
}

func validateStock(stock domain.Stock) error {
	if strings.TrimSpace(stock.TSCode) == "" || strings.TrimSpace(stock.Name) == "" || strings.TrimSpace(stock.Exchange) == "" || strings.TrimSpace(stock.Market) == "" {
		return fmt.Errorf("stock code, name, exchange, and market are required")
	}
	if !stock.ListDate.Valid() {
		return fmt.Errorf("valid list date is required")
	}
	if stock.DelistDate != nil && (!stock.DelistDate.Valid() || stock.DelistDate.String() < stock.ListDate.String()) {
		return fmt.Errorf("delist date must be valid and not precede list date")
	}
	if stock.ListStatus != "L" && stock.ListStatus != "D" && stock.ListStatus != "P" {
		return fmt.Errorf("list status must be L, D, or P")
	}
	if stock.UpdatedAt.IsZero() {
		return fmt.Errorf("updated time is required")
	}
	return nil
}

func validateDailyPrice(item domain.DailyPrice) error {
	if err := validateCodeDate(item.TSCode, item.TradeDate, "stock code"); err != nil {
		return err
	}
	for name, value := range map[string]float64{"open": item.Open, "high": item.High, "low": item.Low, "close": item.Close} {
		if err := validateDecimal(name, value, 20, 6, true); err != nil {
			return err
		}
	}
	if item.High < item.Open || item.High < item.Close || item.Low > item.Open || item.Low > item.Close || item.High < item.Low {
		return fmt.Errorf("daily OHLC values are inconsistent")
	}
	if err := validateDecimal("amount_yuan", item.AmountYuan.Float64(), 24, 4, false); err != nil {
		return err
	}
	if err := validateDecimal("volume_lot", item.VolumeLot, 24, 4, false); err != nil {
		return err
	}
	if err := validateSourceHash(item.SourceHash); err != nil {
		return err
	}
	if item.FetchedAt.IsZero() {
		return fmt.Errorf("fetched time is required")
	}
	return nil
}

func validateAdjFactor(item domain.AdjFactor) error {
	if err := validateCodeDate(item.TSCode, item.TradeDate, "stock code"); err != nil {
		return err
	}
	if err := validateDecimal("adjustment factor", item.Factor, 24, 10, true); err != nil {
		return err
	}
	if err := validateSourceHash(item.SourceHash); err != nil {
		return err
	}
	if item.FetchedAt.IsZero() {
		return fmt.Errorf("fetched time is required")
	}
	return nil
}

func validateCodeDate(code string, date types.TradingDate, label string) error {
	if strings.TrimSpace(code) == "" {
		return fmt.Errorf("%s is required", label)
	}
	if !date.Valid() {
		return fmt.Errorf("valid trading date is required")
	}
	return nil
}

func validateRange(code string, from, through types.TradingDate, label string) error {
	if strings.TrimSpace(code) == "" {
		return fmt.Errorf("%s is required", label)
	}
	if !from.Valid() || !through.Valid() {
		return fmt.Errorf("valid date range is required")
	}
	if from.String() > through.String() {
		return fmt.Errorf("date range start must not follow end")
	}
	return nil
}

func validateDecimal(name string, value float64, precision, scale int, positive bool) error {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return fmt.Errorf("%s must be finite", name)
	}
	if positive && value <= 0 {
		return fmt.Errorf("%s must be positive", name)
	}
	if !positive && value < 0 {
		return fmt.Errorf("%s must not be negative", name)
	}
	formatted := strconv.FormatFloat(math.Abs(value), 'f', -1, 64)
	parts := strings.SplitN(formatted, ".", 2)
	integerDigits := len(strings.TrimLeft(parts[0], "0"))
	if integerDigits == 0 {
		integerDigits = 1
	}
	fractionDigits := 0
	if len(parts) == 2 {
		fractionDigits = len(strings.TrimRight(parts[1], "0"))
	}
	if integerDigits > precision-scale || fractionDigits > scale {
		return fmt.Errorf("%s exceeds DECIMAL(%d,%d) without rounding", name, precision, scale)
	}
	return nil
}

func validateSourceHash(value string) error {
	if len(value) != 64 {
		return fmt.Errorf("source hash must be 64 hexadecimal characters")
	}
	if _, err := hex.DecodeString(value); err != nil {
		return fmt.Errorf("source hash must be 64 hexadecimal characters: %w", err)
	}
	return nil
}

func nullableDate(value *types.TradingDate) (any, error) {
	if value == nil {
		return nil, nil
	}
	if !value.Valid() {
		return nil, fmt.Errorf("date must be valid")
	}
	return value.String(), nil
}

func databaseTimestamp(value time.Time) string {
	return value.UTC().Truncate(time.Microsecond).Format("2006-01-02 15:04:05.000000")
}

func parseDatabaseTimestamp(value string) (time.Time, error) {
	parsed, err := time.ParseInLocation("2006-01-02 15:04:05.000000", value, time.UTC)
	if err != nil {
		return time.Time{}, err
	}
	return parsed, nil
}
