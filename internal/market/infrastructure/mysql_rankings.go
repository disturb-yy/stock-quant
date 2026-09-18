package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/disturb-yy/stock-quant/internal/market"
	marketdomain "github.com/disturb-yy/stock-quant/internal/market/domain"
)

// ReadRankingSnapshot 读取最新交易日股票行情及换手率指标。
func (reader *MySQLOverviewReader) ReadRankingSnapshot(ctx context.Context) (market.RankingSnapshot, error) {
	observations, asOf, err := reader.readRankingObservations(ctx)
	if err != nil {
		return market.RankingSnapshot{}, err
	}
	if len(observations) == 0 {
		return market.RankingSnapshot{}, market.ErrRankingNoData
	}
	seedVersion, err := reader.readSeedVersion(ctx)
	if err != nil {
		return market.RankingSnapshot{}, err
	}
	return market.RankingSnapshot{SeedVersion: seedVersion, AsOf: asOf, Observations: observations}, nil
}

func (reader *MySQLOverviewReader) readRankingObservations(ctx context.Context) ([]marketdomain.RankingObservation, string, error) {
	rows, err := reader.db.QueryContext(ctx, `
		WITH latest_trade AS (
			SELECT MAX(trade_date) AS trade_date FROM daily_bars
		), previous_trade AS (
			SELECT MAX(trade_date) AS trade_date
			FROM daily_bars
			WHERE trade_date < (SELECT trade_date FROM latest_trade)
		)
		SELECT instruments.code, instruments.name,
		       DATE_FORMAT(current_bar.trade_date, '%Y-%m-%d'),
		       CAST(previous_bar.close_price AS CHAR),
		       CAST(current_bar.close_price AS CHAR),
		       CAST(current_bar.turnover_amount AS CHAR),
		       CAST(turnover_rate.metric_value AS CHAR)
		FROM instruments
		CROSS JOIN latest_trade
		INNER JOIN daily_bars AS current_bar
		  ON current_bar.instrument_code = instruments.code
		 AND current_bar.trade_date = latest_trade.trade_date
		CROSS JOIN previous_trade
		INNER JOIN daily_bars AS previous_bar
		  ON previous_bar.instrument_code = instruments.code
		 AND previous_bar.trade_date = previous_trade.trade_date
		LEFT JOIN financial_metrics AS turnover_rate
		  ON turnover_rate.instrument_code = instruments.code
		 AND turnover_rate.metric_date = latest_trade.trade_date
		 AND turnover_rate.metric_name = 'turnover_rate'
		WHERE instruments.status = 'active'
		ORDER BY instruments.code`)
	if err != nil {
		return nil, "", fmt.Errorf("read market ranking observations: %w", err)
	}
	defer rows.Close()

	observations := make([]marketdomain.RankingObservation, 0)
	var asOf string
	for rows.Next() {
		observation, tradeDate, err := scanRankingObservation(rows)
		if err != nil {
			return nil, "", err
		}
		if asOf == "" {
			asOf = tradeDate
		}
		if asOf != tradeDate {
			return nil, "", fmt.Errorf("market ranking observations have mixed trade dates")
		}
		observations = append(observations, observation)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate market ranking observations: %w", err)
	}
	return observations, asOf, nil
}

func scanRankingObservation(rows *sql.Rows) (marketdomain.RankingObservation, string, error) {
	var observation marketdomain.RankingObservation
	var tradeDate, previousClose, turnoverRate sql.NullString
	if err := rows.Scan(
		&observation.InstrumentCode, &observation.InstrumentName, &tradeDate,
		&previousClose, &observation.CurrentClose, &observation.TurnoverAmount, &turnoverRate,
	); err != nil {
		return marketdomain.RankingObservation{}, "", fmt.Errorf("scan market ranking observation: %w", err)
	}
	if !tradeDate.Valid || tradeDate.String == "" {
		return marketdomain.RankingObservation{}, "", errors.New("market ranking trade date is incomplete")
	}
	if previousClose.Valid {
		observation.PreviousClose = previousClose.String
	}
	if turnoverRate.Valid {
		observation.TurnoverRate = turnoverRate.String
	}
	return observation, tradeDate.String, nil
}

var _ market.RankingReader = (*MySQLOverviewReader)(nil)
