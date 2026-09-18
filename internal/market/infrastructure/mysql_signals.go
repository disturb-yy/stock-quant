package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/disturb-yy/stock-quant/internal/market"
	marketdomain "github.com/disturb-yy/stock-quant/internal/market/domain"
)

// ReadSignalSnapshot 读取每只在市股票截至最新交易日的窗口行情。
func (reader *MySQLOverviewReader) ReadSignalSnapshot(ctx context.Context, window int) (market.SignalSnapshot, error) {
	series, err := reader.readSignalSeries(ctx, window)
	if err != nil {
		return market.SignalSnapshot{}, err
	}
	if len(series) == 0 {
		return market.SignalSnapshot{}, errors.New("market signal data is empty")
	}
	seedVersion, err := reader.readSeedVersion(ctx)
	if err != nil {
		return market.SignalSnapshot{}, err
	}
	return market.SignalSnapshot{
		SeedVersion: seedVersion,
		AsOf:        series[0].Bars[len(series[0].Bars)-1].TradeDate,
		Series:      series,
	}, nil
}

func (reader *MySQLOverviewReader) readSignalSeries(ctx context.Context, window int) ([]marketdomain.SignalSeries, error) {
	if window < 1 {
		return nil, errors.New("signal window must be positive")
	}
	rows, err := reader.db.QueryContext(ctx, `
		WITH ranked_bars AS (
			SELECT instrument_code, trade_date, open_price, high_price, low_price,
			       close_price, volume, turnover_amount,
			       ROW_NUMBER() OVER (PARTITION BY instrument_code ORDER BY trade_date DESC) AS bar_number
			FROM daily_bars
			WHERE trade_date <= (SELECT MAX(trade_date) FROM daily_bars)
		)
		SELECT instruments.code, instruments.name,
		       DATE_FORMAT(ranked_bars.trade_date, '%Y-%m-%d'),
		       CAST(ranked_bars.open_price AS CHAR),
		       CAST(ranked_bars.high_price AS CHAR),
		       CAST(ranked_bars.low_price AS CHAR),
		       CAST(ranked_bars.close_price AS CHAR),
		       ranked_bars.volume,
		       CAST(ranked_bars.turnover_amount AS CHAR)
		FROM ranked_bars
		INNER JOIN instruments ON instruments.code = ranked_bars.instrument_code
		WHERE instruments.status = 'active' AND ranked_bars.bar_number <= ?
		ORDER BY instruments.code, ranked_bars.trade_date`, window+1)
	if err != nil {
		return nil, fmt.Errorf("read market signal daily bars: %w", err)
	}
	defer rows.Close()

	seriesByCode := make(map[string]*marketdomain.SignalSeries)
	order := make([]string, 0)
	latestTradeDate := ""
	for rows.Next() {
		var code, name string
		var bar marketdomain.DailyBar
		if err := rows.Scan(&code, &name, &bar.TradeDate, &bar.Open, &bar.High, &bar.Low, &bar.Close, &bar.Volume, &bar.TurnoverAmount); err != nil {
			return nil, fmt.Errorf("scan market signal daily bar: %w", err)
		}
		bar.InstrumentCode = code
		if bar.TradeDate > latestTradeDate {
			latestTradeDate = bar.TradeDate
		}
		series, ok := seriesByCode[code]
		if !ok {
			series = &marketdomain.SignalSeries{InstrumentCode: code, InstrumentName: name, Bars: make([]marketdomain.DailyBar, 0, window+1)}
			seriesByCode[code] = series
			order = append(order, code)
		}
		series.Bars = append(series.Bars, bar)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate market signal daily bars: %w", err)
	}
	series := make([]marketdomain.SignalSeries, 0, len(order))
	for _, code := range order {
		item := *seriesByCode[code]
		// 停牌、刚上市股票可能没有完整窗口；跳过它们，避免一只股票阻断全市场扫描。
		if len(item.Bars) >= window+1 && item.Bars[len(item.Bars)-1].TradeDate == latestTradeDate {
			series = append(series, item)
		}
	}
	return series, nil
}

var _ market.SignalReader = (*MySQLOverviewReader)(nil)
