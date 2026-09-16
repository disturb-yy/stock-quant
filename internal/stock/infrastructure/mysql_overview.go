package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/disturb-yy/stock-quant/internal/stock"
)

// MySQLOverviewReader 从已迁移的股票和行情表读取详情概览。
type MySQLOverviewReader struct {
	db *sql.DB
}

// NewMySQLOverviewReader 创建股票详情 MySQL 读取器。
func NewMySQLOverviewReader(db *sql.DB) (*MySQLOverviewReader, error) {
	if db == nil {
		return nil, errors.New("stock overview database connection is required")
	}
	return &MySQLOverviewReader{db: db}, nil
}

// ReadStockOverview 聚合一只股票的身份、行情、指标和走势图。
func (reader *MySQLOverviewReader) ReadStockOverview(ctx context.Context, symbol string) (stock.OverviewSnapshot, error) {
	snapshot, err := reader.readIdentity(ctx, symbol)
	if err != nil {
		return stock.OverviewSnapshot{}, err
	}
	if err := reader.readQuote(ctx, symbol, &snapshot); err != nil {
		return stock.OverviewSnapshot{}, err
	}
	if err := reader.readMetrics(ctx, symbol, snapshot.Quote.AsOf, &snapshot); err != nil {
		return stock.OverviewSnapshot{}, err
	}
	if err := reader.readSparkline(ctx, symbol, &snapshot); err != nil {
		return stock.OverviewSnapshot{}, err
	}
	return snapshot, nil
}

func (reader *MySQLOverviewReader) readIdentity(ctx context.Context, symbol string) (stock.OverviewSnapshot, error) {
	var snapshot stock.OverviewSnapshot
	err := reader.db.QueryRowContext(ctx, `
		SELECT instruments.code, instruments.name,
		       COALESCE(GROUP_CONCAT(DISTINCT sector_categories.name ORDER BY sector_categories.name SEPARATOR '、'), '')
		FROM instruments
		LEFT JOIN sector_memberships
		  ON sector_memberships.instrument_code = instruments.code
		LEFT JOIN sector_categories
		  ON sector_categories.code = sector_memberships.sector_code
		WHERE instruments.code = ?
		GROUP BY instruments.code, instruments.name`, symbol).Scan(&snapshot.Symbol, &snapshot.Name, &snapshot.Industry)
	if errors.Is(err, sql.ErrNoRows) {
		return stock.OverviewSnapshot{}, stock.ErrInstrumentNotFound
	}
	if err != nil {
		return stock.OverviewSnapshot{}, fmt.Errorf("read stock identity: %w", err)
	}
	return snapshot, nil
}

func (reader *MySQLOverviewReader) readQuote(ctx context.Context, symbol string, snapshot *stock.OverviewSnapshot) error {
	var asOf, last, change, changePct sql.NullString
	err := reader.db.QueryRowContext(ctx, `
		WITH latest_bar AS (
			SELECT instrument_code, MAX(trade_date) AS trade_date
			FROM daily_bars
			WHERE instrument_code = ?
			GROUP BY instrument_code
		), previous_bar AS (
			SELECT MAX(trade_date) AS trade_date
			FROM daily_bars
			WHERE instrument_code = ?
			  AND trade_date < (SELECT trade_date FROM latest_bar)
		)
		SELECT DATE_FORMAT(current_bar.trade_date, '%Y-%m-%d'),
		       CAST(ROUND(current_bar.close_price, 2) AS CHAR),
		       CAST(ROUND(current_bar.close_price - previous.close_price, 2) AS CHAR),
		       CAST(ROUND((current_bar.close_price - previous.close_price) / NULLIF(previous.close_price, 0) * 100, 2) AS CHAR)
		FROM latest_bar
		INNER JOIN daily_bars AS current_bar
		  ON current_bar.instrument_code = latest_bar.instrument_code
		 AND current_bar.trade_date = latest_bar.trade_date
		CROSS JOIN previous_bar
		INNER JOIN daily_bars AS previous
		  ON previous.instrument_code = current_bar.instrument_code
		 AND previous.trade_date = previous_bar.trade_date`, symbol, symbol).
		Scan(&asOf, &last, &change, &changePct)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w for %q", stock.ErrQuoteUnavailable, symbol)
	}
	if err != nil {
		return fmt.Errorf("read stock quote: %w", err)
	}
	if !asOf.Valid || !last.Valid || !change.Valid || !changePct.Valid {
		return fmt.Errorf("%w for %q", stock.ErrQuoteUnavailable, symbol)
	}
	snapshot.Quote = stock.QuoteSnapshot{Last: last.String, Change: change.String, ChangePct: changePct.String, AsOf: asOf.String}
	return nil
}

func (reader *MySQLOverviewReader) readMetrics(ctx context.Context, symbol, asOf string, snapshot *stock.OverviewSnapshot) error {
	if err := reader.readDailyBasic(ctx, symbol, asOf, snapshot); err != nil {
		return err
	}
	return reader.readFinancialMetrics(ctx, symbol, asOf, snapshot)
}

func (reader *MySQLOverviewReader) readDailyBasic(ctx context.Context, symbol, asOf string, snapshot *stock.OverviewSnapshot) error {
	var metricDate, marketCap, pb sql.NullString
	err := reader.db.QueryRowContext(ctx, `
		SELECT DATE_FORMAT(trade_date, '%Y-%m-%d'),
		       CAST(ROUND(market_cap, 2) AS CHAR), CAST(ROUND(pb, 2) AS CHAR)
		FROM daily_basic
		WHERE instrument_code = ? AND trade_date <= ?
		ORDER BY trade_date DESC LIMIT 1`, symbol, asOf).Scan(&metricDate, &marketCap, &pb)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read stock daily basic: %w", err)
	}
	if !metricDate.Valid {
		return nil
	}
	snapshot.MarketCap = stock.MetricSnapshot{Value: nullableString(marketCap), AsOf: metricDate.String, Basis: "latest_daily_basic"}
	snapshot.PB = stock.MetricSnapshot{Value: nullableString(pb), AsOf: metricDate.String, Basis: "latest_daily_basic"}
	return nil
}

func (reader *MySQLOverviewReader) readFinancialMetrics(ctx context.Context, symbol, asOf string, snapshot *stock.OverviewSnapshot) error {
	rows, err := reader.db.QueryContext(ctx, `
		SELECT metric_name, DATE_FORMAT(metric_date, '%Y-%m-%d'), basis, CAST(metric_value AS CHAR)
		FROM financial_metrics
		WHERE instrument_code = ? AND metric_date <= ? AND metric_name IN ('pe_ttm', 'roe')
		ORDER BY metric_date DESC`, symbol, asOf)
	if err != nil {
		return fmt.Errorf("read stock financial metrics: %w", err)
	}
	defer rows.Close()
	seen := map[string]bool{}
	for rows.Next() {
		var name, metricDate, basis, value string
		if err := rows.Scan(&name, &metricDate, &basis, &value); err != nil {
			return fmt.Errorf("scan stock financial metric: %w", err)
		}
		if seen[name] {
			continue
		}
		metric := stock.MetricSnapshot{Value: stringPointer(value), AsOf: metricDate, Basis: basis}
		if name == "pe_ttm" {
			snapshot.PETTM = metric
		} else if name == "roe" {
			snapshot.ROE = metric
		}
		seen[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate stock financial metrics: %w", err)
	}
	return nil
}

func (reader *MySQLOverviewReader) readSparkline(ctx context.Context, symbol string, snapshot *stock.OverviewSnapshot) error {
	rows, err := reader.db.QueryContext(ctx, `
		SELECT DATE_FORMAT(trade_date, '%Y-%m-%d'),
		       CAST(ROUND(open_price, 2) AS CHAR),
		       CAST(ROUND(high_price, 2) AS CHAR),
		       CAST(ROUND(low_price, 2) AS CHAR),
		       CAST(ROUND(close_price, 2) AS CHAR)
		FROM daily_bars
		WHERE instrument_code = ?
		ORDER BY trade_date DESC LIMIT 20`, symbol)
	if err != nil {
		return fmt.Errorf("read stock sparkline: %w", err)
	}
	defer rows.Close()
	points := make([]stock.SparklinePoint, 0, 20)
	for rows.Next() {
		var point stock.SparklinePoint
		if err := rows.Scan(&point.TradeDate, &point.Open, &point.High, &point.Low, &point.Close); err != nil {
			return fmt.Errorf("scan stock sparkline point: %w", err)
		}
		points = append(points, point)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate stock sparkline: %w", err)
	}
	if len(points) < 20 {
		snapshot.Sparkline = []stock.SparklinePoint{}
		return nil
	}
	for left, right := 0, len(points)-1; left < right; left, right = left+1, right-1 {
		points[left], points[right] = points[right], points[left]
	}
	snapshot.Sparkline = points
	return nil
}

func nullableString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return stringPointer(value.String)
}

func stringPointer(value string) *string {
	return &value
}

var _ stock.OverviewReader = (*MySQLOverviewReader)(nil)
