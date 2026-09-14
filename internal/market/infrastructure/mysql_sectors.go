package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/disturb-yy/stock-quant/internal/market"
	marketdomain "github.com/disturb-yy/stock-quant/internal/market/domain"
)

// ReadSectorSnapshot 读取最新交易日的行业成分前后收盘价。
func (reader *MySQLOverviewReader) ReadSectorSnapshot(ctx context.Context) (market.SectorSnapshot, error) {
	components, err := reader.readSectorComponents(ctx)
	if err != nil {
		return market.SectorSnapshot{}, err
	}
	seedVersion, err := reader.readSeedVersion(ctx)
	if err != nil {
		return market.SectorSnapshot{}, err
	}
	if len(components) == 0 {
		return market.SectorSnapshot{}, errors.New("market sector components are empty")
	}
	return market.SectorSnapshot{
		SeedVersion: seedVersion,
		AsOf:        components[0].TradeDate,
		Components:  components,
	}, nil
}

func (reader *MySQLOverviewReader) readSectorComponents(ctx context.Context) ([]marketdomain.SectorComponent, error) {
	rows, err := reader.db.QueryContext(ctx, `
		WITH latest_trade AS (
			SELECT MAX(trade_date) AS trade_date FROM daily_bars
		), previous_trade AS (
			SELECT MAX(trade_date) AS trade_date
			FROM daily_bars
			WHERE trade_date < (SELECT trade_date FROM latest_trade)
		)
		SELECT sectors.code, sectors.name,
		       instruments.code, instruments.name,
		       DATE_FORMAT(current_bar.trade_date, '%Y-%m-%d'),
		       CAST(previous_bar.close_price AS CHAR),
		       CAST(current_bar.close_price AS CHAR)
		FROM sector_categories AS sectors
		INNER JOIN sector_memberships AS memberships
		  ON memberships.sector_code = sectors.code
		INNER JOIN instruments
		  ON instruments.code = memberships.instrument_code
		CROSS JOIN latest_trade
		CROSS JOIN previous_trade
		LEFT JOIN daily_bars AS current_bar
		  ON current_bar.instrument_code = instruments.code
		 AND current_bar.trade_date = latest_trade.trade_date
		LEFT JOIN daily_bars AS previous_bar
		  ON previous_bar.instrument_code = instruments.code
		 AND previous_bar.trade_date = previous_trade.trade_date
		ORDER BY sectors.code, instruments.code`)
	if err != nil {
		return nil, fmt.Errorf("read market sector components: %w", err)
	}
	defer rows.Close()

	components := make([]marketdomain.SectorComponent, 0)
	for rows.Next() {
		component, err := scanSectorComponent(rows)
		if err != nil {
			return nil, err
		}
		components = append(components, component)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate market sector components: %w", err)
	}
	return components, nil
}

func scanSectorComponent(rows *sql.Rows) (marketdomain.SectorComponent, error) {
	var component marketdomain.SectorComponent
	var tradeDate, previousClose, currentClose sql.NullString
	if err := rows.Scan(
		&component.SectorCode, &component.SectorName,
		&component.InstrumentCode, &component.InstrumentName,
		&tradeDate, &previousClose, &currentClose,
	); err != nil {
		return marketdomain.SectorComponent{}, fmt.Errorf("scan market sector component: %w", err)
	}
	if !tradeDate.Valid || !previousClose.Valid || !currentClose.Valid {
		return marketdomain.SectorComponent{}, errors.New("market sector component data is incomplete")
	}
	component.TradeDate = tradeDate.String
	component.PreviousClose = previousClose.String
	component.CurrentClose = currentClose.String
	return component, nil
}

var _ market.SectorReader = (*MySQLOverviewReader)(nil)
