package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/disturb-yy/stock-quant/internal/market"
	marketdomain "github.com/disturb-yy/stock-quant/internal/market/domain"
)

// ReadStockBars 读取股票身份、指定范围日线、复权因子和可选基准序列。
func (reader *MySQLOverviewReader) ReadStockBars(ctx context.Context, request market.BarsRequest) (market.BarsSnapshot, error) {
	snapshot, err := reader.readBarsIdentity(ctx, request.Symbol)
	if err != nil {
		return market.BarsSnapshot{}, err
	}
	snapshot.Bars, err = reader.readBars(ctx, request)
	if err != nil {
		return market.BarsSnapshot{}, err
	}
	if request.Benchmark != "" {
		snapshot.BenchmarkCode = request.Benchmark
		snapshot.BenchmarkName, snapshot.Benchmark, err = reader.readBenchmark(ctx, request.Benchmark)
		if err != nil {
			return market.BarsSnapshot{}, err
		}
	}
	snapshot.SeedVersion, err = reader.readSeedVersion(ctx)
	if err != nil {
		return market.BarsSnapshot{}, err
	}
	return snapshot, nil
}

func (reader *MySQLOverviewReader) readBarsIdentity(ctx context.Context, symbol string) (market.BarsSnapshot, error) {
	var snapshot market.BarsSnapshot
	err := reader.db.QueryRowContext(ctx, `SELECT code, name FROM instruments WHERE code = ?`, symbol).Scan(&snapshot.Symbol, &snapshot.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return market.BarsSnapshot{}, market.ErrBarsInstrumentNotFound
	}
	if err != nil {
		return market.BarsSnapshot{}, fmt.Errorf("read stock bars identity: %w", err)
	}
	return snapshot, nil
}

func (reader *MySQLOverviewReader) readBars(ctx context.Context, request market.BarsRequest) ([]marketdomain.BarWithAdjustment, error) {
	query, args, reverse, err := barsQuery(request)
	if err != nil {
		return nil, err
	}
	rows, err := reader.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("read stock daily bars: %w", err)
	}
	defer rows.Close()
	bars := make([]marketdomain.BarWithAdjustment, 0)
	for rows.Next() {
		bar, err := scanBarWithAdjustment(rows, request.Symbol)
		if err != nil {
			return nil, err
		}
		bars = append(bars, bar)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stock daily bars: %w", err)
	}
	if reverse {
		reverseBars(bars)
	}
	return bars, nil
}

func barsQuery(request market.BarsRequest) (string, []any, bool, error) {
	base := `
		SELECT DATE_FORMAT(daily_bars.trade_date, '%Y-%m-%d'),
		       CAST(daily_bars.open_price AS CHAR), CAST(daily_bars.high_price AS CHAR),
		       CAST(daily_bars.low_price AS CHAR), CAST(daily_bars.close_price AS CHAR),
		       daily_bars.volume, CAST(daily_bars.turnover_amount AS CHAR),
		       CAST(adjustment_factors.qfq_factor AS CHAR), CAST(adjustment_factors.hfq_factor AS CHAR)
		FROM daily_bars
		LEFT JOIN daily_adjustment_factors AS adjustment_factors
		  ON adjustment_factors.instrument_code = daily_bars.instrument_code
		 AND adjustment_factors.trade_date = daily_bars.trade_date
		WHERE daily_bars.instrument_code = ?`
	if request.From != "" && request.To != "" {
		return base + ` AND daily_bars.trade_date BETWEEN ? AND ? ORDER BY daily_bars.trade_date`, []any{request.Symbol, request.From, request.To}, false, nil
	}
	if request.Range == "all" {
		return base + ` ORDER BY daily_bars.trade_date`, []any{request.Symbol}, false, nil
	}
	limit, ok := map[string]int{"20d": 20, "60d": 60, "120d": 120}[request.Range]
	if !ok {
		return "", nil, false, fmt.Errorf("unsupported stock bars range %q", request.Range)
	}
	return base + ` ORDER BY daily_bars.trade_date DESC LIMIT ?`, []any{request.Symbol, limit}, true, nil
}

func scanBarWithAdjustment(rows *sql.Rows, symbol string) (marketdomain.BarWithAdjustment, error) {
	var bar marketdomain.BarWithAdjustment
	var tradeDate, open, high, low, close, turnover string
	var volume int64
	var qfqFactor, hfqFactor sql.NullString
	if err := rows.Scan(&tradeDate, &open, &high, &low, &close, &volume, &turnover, &qfqFactor, &hfqFactor); err != nil {
		return marketdomain.BarWithAdjustment{}, fmt.Errorf("scan stock daily bar: %w", err)
	}
	bar.Bar = marketdomain.DailyBar{InstrumentCode: symbol, TradeDate: tradeDate, Open: open, High: high, Low: low, Close: close, Volume: volume, TurnoverAmount: turnover}
	if qfqFactor.Valid {
		bar.QFQFactor = qfqFactor.String
	}
	if hfqFactor.Valid {
		bar.HFQFactor = hfqFactor.String
	}
	return bar, nil
}

func (reader *MySQLOverviewReader) readBenchmark(ctx context.Context, code string) (string, []marketdomain.BenchmarkBar, error) {
	rows, err := reader.db.QueryContext(ctx, `
		SELECT name, DATE_FORMAT(trade_date, '%Y-%m-%d'), CAST(close_price AS CHAR)
		FROM index_snapshots WHERE code = ? ORDER BY trade_date`, code)
	if err != nil {
		return "", nil, fmt.Errorf("read stock bars benchmark: %w", err)
	}
	defer rows.Close()
	name := ""
	points := make([]marketdomain.BenchmarkBar, 0)
	for rows.Next() {
		var point marketdomain.BenchmarkBar
		if err := rows.Scan(&name, &point.TradeDate, &point.Close); err != nil {
			return "", nil, fmt.Errorf("scan stock bars benchmark: %w", err)
		}
		points = append(points, point)
	}
	if err := rows.Err(); err != nil {
		return "", nil, fmt.Errorf("iterate stock bars benchmark: %w", err)
	}
	if len(points) == 0 {
		return "", nil, fmt.Errorf("%w: %s", market.ErrBarsBenchmarkUnavailable, code)
	}
	return name, points, nil
}

func reverseBars(bars []marketdomain.BarWithAdjustment) {
	for left, right := 0, len(bars)-1; left < right; left, right = left+1, right-1 {
		bars[left], bars[right] = bars[right], bars[left]
	}
}

var _ market.BarsReader = (*MySQLOverviewReader)(nil)
