package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/disturb-yy/stock-quant/internal/demo"
	"github.com/disturb-yy/stock-quant/internal/stock"
	stockdomain "github.com/disturb-yy/stock-quant/internal/stock/domain"
)

// MySQLValuationReader 从估值快照和既有行业成员关系读取股票估值研究数据。
type MySQLValuationReader struct {
	db *sql.DB
}

// NewMySQLValuationReader 创建股票估值 MySQL 读取器。
func NewMySQLValuationReader(db *sql.DB) (*MySQLValuationReader, error) {
	if db == nil {
		return nil, errors.New("stock valuation database connection is required")
	}
	return &MySQLValuationReader{db: db}, nil
}

// ReadStockValuation 读取目标股票、行业映射和同日可比成员的完整原始快照。
func (reader *MySQLValuationReader) ReadStockValuation(ctx context.Context, request stock.ValuationRequest) (stock.ValuationSnapshot, error) {
	snapshot, err := reader.readValuationIdentity(ctx, request.Symbol)
	if err != nil {
		return stock.ValuationSnapshot{}, err
	}
	snapshot.SeedVersion, snapshot.SourceAsOf, err = reader.readValuationSource(ctx)
	if err != nil {
		return stock.ValuationSnapshot{}, err
	}
	snapshot.Observations, err = reader.readValuationObservations(ctx, request.Symbol)
	if err != nil {
		return stock.ValuationSnapshot{}, err
	}
	snapshot.Industries, err = reader.readValuationIndustries(ctx, request.Symbol)
	if err != nil {
		return stock.ValuationSnapshot{}, err
	}
	return snapshot, nil
}

func (reader *MySQLValuationReader) readValuationIdentity(ctx context.Context, symbol string) (stock.ValuationSnapshot, error) {
	var snapshot stock.ValuationSnapshot
	err := reader.db.QueryRowContext(ctx, `SELECT code, name FROM instruments WHERE code = ?`, symbol).Scan(&snapshot.Symbol, &snapshot.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return stock.ValuationSnapshot{}, stock.ErrValuationInstrumentNotFound
	}
	if err != nil {
		return stock.ValuationSnapshot{}, fmt.Errorf("read stock valuation identity: %w", err)
	}
	return snapshot, nil
}

func (reader *MySQLValuationReader) readValuationSource(ctx context.Context) (string, string, error) {
	var seedVersion, asOf string
	err := reader.db.QueryRowContext(ctx, `
        SELECT seed_version, DATE_FORMAT(as_of, '%Y-%m-%d')
        FROM demo_seed_metadata WHERE seed_name = ?`, demo.SeedName).Scan(&seedVersion, &asOf)
	if err != nil {
		return "", "", fmt.Errorf("read stock valuation source: %w", err)
	}
	return seedVersion, asOf, nil
}

func (reader *MySQLValuationReader) readValuationObservations(ctx context.Context, symbol string) ([]stockdomain.ValuationObservation, error) {
	rows, err := reader.db.QueryContext(ctx, `
        SELECT DATE_FORMAT(trade_date, '%Y-%m-%d'),
               CAST(pe_ttm AS CHAR), pe_ttm_basis,
               CAST(pb AS CHAR), pb_basis,
               CAST(ps_ttm AS CHAR), ps_ttm_basis
        FROM stock_valuation_snapshots
        WHERE instrument_code = ?
        ORDER BY trade_date ASC`, symbol)
	if err != nil {
		return nil, fmt.Errorf("read stock valuation observations: %w", err)
	}
	defer rows.Close()
	observations := make([]stockdomain.ValuationObservation, 0)
	for rows.Next() {
		observation, err := scanValuationObservation(rows, symbol)
		if err != nil {
			return nil, err
		}
		observations = append(observations, observation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stock valuation observations: %w", err)
	}
	return observations, nil
}

func (reader *MySQLValuationReader) readValuationIndustries(ctx context.Context, symbol string) ([]stock.ValuationIndustrySnapshot, error) {
	rows, err := reader.db.QueryContext(ctx, `
        SELECT sectors.code, sectors.name, members.code, members.name,
               DATE_FORMAT(valuation.trade_date, '%Y-%m-%d'),
               CAST(valuation.pe_ttm AS CHAR), valuation.pe_ttm_basis,
               CAST(valuation.pb AS CHAR), valuation.pb_basis,
               CAST(valuation.ps_ttm AS CHAR), valuation.ps_ttm_basis
        FROM sector_memberships AS target_membership
        INNER JOIN sector_categories AS sectors
          ON sectors.code = target_membership.sector_code
        INNER JOIN sector_memberships AS member_membership
          ON member_membership.sector_code = target_membership.sector_code
        INNER JOIN instruments AS members
          ON members.code = member_membership.instrument_code
        LEFT JOIN stock_valuation_snapshots AS valuation
          ON valuation.instrument_code = members.code
        WHERE target_membership.instrument_code = ?
        ORDER BY sectors.code, members.code, valuation.trade_date`, symbol)
	if err != nil {
		return nil, fmt.Errorf("read stock valuation industries: %w", err)
	}
	defer rows.Close()

	industries := make([]stock.ValuationIndustrySnapshot, 0)
	industryIndexes := make(map[string]int)
	memberIndexes := make(map[string]map[string]int)
	for rows.Next() {
		var industryCode, industryName, memberCode, memberName string
		var tradeDate, pe, peBasis, pb, pbBasis, ps, psBasis sql.NullString
		if err := rows.Scan(&industryCode, &industryName, &memberCode, &memberName, &tradeDate, &pe, &peBasis, &pb, &pbBasis, &ps, &psBasis); err != nil {
			return nil, fmt.Errorf("scan stock valuation industry: %w", err)
		}
		industryIndex, ok := industryIndexes[industryCode]
		if !ok {
			industryIndex = len(industries)
			industryIndexes[industryCode] = industryIndex
			memberIndexes[industryCode] = make(map[string]int)
			industries = append(industries, stock.ValuationIndustrySnapshot{Code: industryCode, Name: industryName, Members: make([]stock.ValuationMemberSnapshot, 0)})
		}
		memberIndex, ok := memberIndexes[industryCode][memberCode]
		if !ok {
			memberIndex = len(industries[industryIndex].Members)
			memberIndexes[industryCode][memberCode] = memberIndex
			industries[industryIndex].Members = append(industries[industryIndex].Members, stock.ValuationMemberSnapshot{InstrumentCode: memberCode, Name: memberName, Observations: make([]stockdomain.ValuationObservation, 0)})
		}
		if !tradeDate.Valid {
			continue
		}
		observation := stockdomain.ValuationObservation{
			InstrumentCode: memberCode,
			AsOf:           tradeDate.String,
			PETTM:          scanValuationMetric(tradeDate.String, pe, peBasis),
			PB:             scanValuationMetric(tradeDate.String, pb, pbBasis),
			PSTTM:          scanValuationMetric(tradeDate.String, ps, psBasis),
		}
		industries[industryIndex].Members[memberIndex].Observations = append(industries[industryIndex].Members[memberIndex].Observations, observation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stock valuation industries: %w", err)
	}
	return industries, nil
}

func scanValuationObservation(rows *sql.Rows, symbol string) (stockdomain.ValuationObservation, error) {
	var tradeDate, pe, peBasis, pb, pbBasis, ps, psBasis sql.NullString
	if err := rows.Scan(&tradeDate, &pe, &peBasis, &pb, &pbBasis, &ps, &psBasis); err != nil {
		return stockdomain.ValuationObservation{}, fmt.Errorf("scan stock valuation observation: %w", err)
	}
	if !tradeDate.Valid {
		return stockdomain.ValuationObservation{}, errors.New("stock valuation observation date is missing")
	}
	return stockdomain.ValuationObservation{
		InstrumentCode: symbol,
		AsOf:           tradeDate.String,
		PETTM:          scanValuationMetric(tradeDate.String, pe, peBasis),
		PB:             scanValuationMetric(tradeDate.String, pb, pbBasis),
		PSTTM:          scanValuationMetric(tradeDate.String, ps, psBasis),
	}, nil
}

func scanValuationMetric(asOf string, value, basis sql.NullString) stockdomain.ValuationMetricObservation {
	metric := stockdomain.ValuationMetricObservation{AsOf: asOf}
	if value.Valid {
		metric.Value = &value.String
		if basis.Valid {
			metric.Basis = basis.String
		}
	}
	return metric
}

var _ stock.ValuationReader = (*MySQLValuationReader)(nil)
