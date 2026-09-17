// Package infrastructure 提供行情领域的 MySQL 查询适配。
package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/disturb-yy/stock-quant/internal/demo"
	"github.com/disturb-yy/stock-quant/internal/market"
	marketdomain "github.com/disturb-yy/stock-quant/internal/market/domain"
)

// MySQLOverviewReader 从已迁移的行情表读取市场概览。
type MySQLOverviewReader struct {
	db           *sql.DB
	metadataName string
}

// NewMySQLOverviewReader 创建 MySQL 市场概览读取器。
func NewMySQLOverviewReader(db *sql.DB, metadataNames ...string) (*MySQLOverviewReader, error) {
	if db == nil {
		return nil, errors.New("market overview database connection is required")
	}
	metadataName := demo.SeedName
	if len(metadataNames) > 0 && metadataNames[0] != "" {
		metadataName = metadataNames[0]
	}
	return &MySQLOverviewReader{db: db, metadataName: metadataName}, nil
}

// ReadOverviewSnapshot 读取最新交易日的四个指数和市场宽度。
func (reader *MySQLOverviewReader) ReadOverviewSnapshot(ctx context.Context) (market.OverviewSnapshot, error) {
	indices, err := reader.readIndices(ctx)
	if err != nil {
		return market.OverviewSnapshot{}, err
	}
	breadth, err := reader.readBreadth(ctx)
	if err != nil {
		return market.OverviewSnapshot{}, err
	}
	seedVersion, err := reader.readSeedVersion(ctx)
	if err != nil {
		return market.OverviewSnapshot{}, err
	}
	if len(indices) == 0 {
		return market.OverviewSnapshot{}, fmt.Errorf("market index snapshots are empty")
	}
	return market.OverviewSnapshot{
		SeedVersion: seedVersion,
		AsOf:        breadth.TradeDate,
		ObservedAt:  indices[0].ObservedAt,
		Indices:     indices,
		Breadth:     breadth,
	}, nil
}

func (reader *MySQLOverviewReader) readIndices(ctx context.Context) ([]marketdomain.IndexSnapshot, error) {
	rows, err := reader.db.QueryContext(ctx, `
		SELECT code, name, DATE_FORMAT(trade_date, '%Y-%m-%d'),
		       DATE_FORMAT(observed_at, '%Y-%m-%dT%H:%i:%sZ'),
		       CAST(close_price AS DECIMAL(20,2)),
		       CAST(change_amount AS DECIMAL(20,2)),
		       CAST(change_percent AS DECIMAL(20,2))
        FROM index_snapshots
        WHERE trade_date = (SELECT MAX(trade_date) FROM index_snapshots)
        ORDER BY code`)
	if err != nil {
		return nil, fmt.Errorf("read market index snapshots: %w", err)
	}
	defer rows.Close()
	indices := make([]marketdomain.IndexSnapshot, 0, 4)
	for rows.Next() {
		var index marketdomain.IndexSnapshot
		if err := rows.Scan(&index.Code, &index.Name, &index.TradeDate, &index.ObservedAt, &index.Close, &index.Change, &index.ChangePercent); err != nil {
			return nil, fmt.Errorf("scan market index snapshot: %w", err)
		}
		indices = append(indices, index)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate market index snapshots: %w", err)
	}
	return indices, nil
}

func (reader *MySQLOverviewReader) readBreadth(ctx context.Context) (marketdomain.MarketBreadth, error) {
	var breadth marketdomain.MarketBreadth
	err := reader.db.QueryRowContext(ctx, `
		SELECT DATE_FORMAT(current_bars.trade_date, '%Y-%m-%d'),
               SUM(CASE WHEN current_bars.close_price > previous_bars.close_price THEN 1 ELSE 0 END),
               SUM(CASE WHEN current_bars.close_price < previous_bars.close_price THEN 1 ELSE 0 END),
               SUM(CASE WHEN current_bars.close_price = previous_bars.close_price THEN 1 ELSE 0 END),
			   CAST(ROUND(SUM(current_bars.turnover_amount), 2) AS DECIMAL(20,2))
        FROM daily_bars AS current_bars
        INNER JOIN daily_bars AS previous_bars
          ON previous_bars.instrument_code = current_bars.instrument_code
         AND previous_bars.trade_date = (
             SELECT MAX(trade_date) FROM daily_bars
             WHERE trade_date < (SELECT MAX(trade_date) FROM daily_bars)
         )
        WHERE current_bars.trade_date = (SELECT MAX(trade_date) FROM daily_bars)
        GROUP BY current_bars.trade_date`).Scan(
		&breadth.TradeDate, &breadth.Advancing, &breadth.Declining, &breadth.Unchanged, &breadth.TurnoverAmount)
	if errors.Is(err, sql.ErrNoRows) {
		return marketdomain.MarketBreadth{}, fmt.Errorf("market breadth data is incomplete: %w", err)
	}
	if err != nil {
		return marketdomain.MarketBreadth{}, fmt.Errorf("read market breadth: %w", err)
	}
	return breadth, nil
}

func (reader *MySQLOverviewReader) readSeedVersion(ctx context.Context) (string, error) {
	var seedVersion string
	err := reader.db.QueryRowContext(ctx, `
        SELECT seed_version FROM demo_seed_metadata WHERE seed_name = ?`, reader.metadataName).Scan(&seedVersion)
	if err != nil {
		return "", fmt.Errorf("read market overview seed metadata: %w", err)
	}
	return seedVersion, nil
}
