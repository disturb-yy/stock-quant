package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/disturb-yy/stock-quant/internal/screener/domain"
)

// MySQLReader 复用已有真实读模型读取选股执行快照。
type MySQLReader struct {
	db           *sql.DB
	source       domain.Source
	metadataName string
}

func NewMySQLReader(db *sql.DB, source domain.Source, metadataNames ...string) (*MySQLReader, error) {
	if db == nil {
		return nil, errors.New("screener database connection is required")
	}
	if strings.TrimSpace(source.Mode) == "" || strings.TrimSpace(source.Provider) == "" {
		return nil, errors.New("screener source is required")
	}
	metadataName := "fnd-003-demo"
	if len(metadataNames) > 0 && metadataNames[0] != "" {
		metadataName = metadataNames[0]
	}
	return &MySQLReader{db: db, source: source, metadataName: metadataName}, nil
}

// ReadSnapshot 一次批量读取所有已发布字段，避免按股票 N+1 查询。
func (reader *MySQLReader) ReadSnapshot(ctx context.Context, fieldIDs []string) (domain.ExecutionInput, error) {
	dates, err := reader.readSnapshotDates(ctx)
	if err != nil {
		return domain.ExecutionInput{}, err
	}
	if dates.market == "" {
		return domain.ExecutionInput{}, errors.New("screener market snapshot is unavailable")
	}
	for _, fieldID := range fieldIDs {
		if fieldDate(dates, fieldID) == "" {
			return domain.ExecutionInput{}, fmt.Errorf("screener field %q snapshot is unavailable", fieldID)
		}
	}
	source := reader.source
	if err := reader.db.QueryRowContext(ctx, `
		SELECT seed_version, provider, DATE_FORMAT(as_of, '%Y-%m-%d')
		FROM demo_seed_metadata WHERE seed_name = ?`, reader.metadataName).Scan(&source.SeedVersion, &source.Provider, &source.AsOf); err != nil {
		return domain.ExecutionInput{}, fmt.Errorf("read screener source metadata: %w", err)
	}
	candidates, err := reader.readCandidates(ctx, dates)
	if err != nil {
		return domain.ExecutionInput{}, err
	}
	if len(candidates) == 0 {
		return domain.ExecutionInput{}, errors.New("screener active universe is empty")
	}
	if err := reader.readIndustries(ctx, candidates); err != nil {
		return domain.ExecutionInput{}, err
	}
	return domain.ExecutionInput{
		Universe: domain.Universe{ID: domain.ActiveAShareUniverse, Name: domain.ActiveAShareName},
		Eligible: candidates,
		Snapshot: domain.Snapshot{
			AsOf: source.AsOf,
			FieldAsOf: map[string]string{
				"market.market_cap": dates.basic, "market.turnover_rate": dates.market,
				"technical.close": dates.market, "technical.volume": dates.market, "technical.turnover_amount": dates.market,
				"valuation.pe_ttm": dates.valuation, "valuation.pb": dates.valuation, "valuation.ps_ttm": dates.valuation,
				"fundamental.revenue": dates.financial, "fundamental.net_profit": dates.financial,
				"fundamental.total_assets": dates.financial, "fundamental.total_liabilities": dates.financial,
				"fundamental.total_equity": dates.financial, "fundamental.operating_cash_flow": dates.financial,
				"fundamental.roe_pct": dates.roe,
			},
			DefinitionVersions: map[string]string{},
		},
		Source: source,
	}, nil
}

func fieldDate(dates snapshotDates, fieldID string) string {
	switch fieldID {
	case "market.market_cap":
		return dates.basic
	case "market.turnover_rate", "technical.close", "technical.volume", "technical.turnover_amount":
		return dates.market
	case "valuation.pe_ttm", "valuation.pb", "valuation.ps_ttm":
		return dates.valuation
	case "fundamental.revenue", "fundamental.net_profit", "fundamental.total_assets", "fundamental.total_liabilities", "fundamental.total_equity", "fundamental.operating_cash_flow":
		return dates.financial
	case "fundamental.roe_pct":
		return dates.roe
	default:
		return ""
	}
}

type snapshotDates struct {
	market, basic, financial, valuation, roe string
}

func (reader *MySQLReader) readSnapshotDates(ctx context.Context) (snapshotDates, error) {
	var dates snapshotDates
	var marketDate, basicDate, financialDate, valuationDate, roeDate sql.NullString
	err := reader.db.QueryRowContext(ctx, `
		SELECT
			DATE_FORMAT((SELECT MAX(trade_date) FROM daily_bars), '%Y-%m-%d'),
			DATE_FORMAT((SELECT MAX(trade_date) FROM daily_basic), '%Y-%m-%d'),
			DATE_FORMAT((SELECT MAX(period_end) FROM stock_financial_reports WHERE period = 'annual'), '%Y-%m-%d'),
			DATE_FORMAT((SELECT MAX(trade_date) FROM stock_valuation_snapshots), '%Y-%m-%d'),
			DATE_FORMAT((SELECT MAX(metric_date) FROM financial_metrics WHERE metric_name = 'roe'), '%Y-%m-%d')`).Scan(&marketDate, &basicDate, &financialDate, &valuationDate, &roeDate)
	if err != nil {
		return snapshotDates{}, fmt.Errorf("read screener snapshot dates: %w", err)
	}
	dates.market, dates.basic, dates.financial, dates.valuation, dates.roe = nullString(marketDate), nullString(basicDate), nullString(financialDate), nullString(valuationDate), nullString(roeDate)
	return dates, nil
}

func (reader *MySQLReader) readCandidates(ctx context.Context, dates snapshotDates) ([]domain.Candidate, error) {
	rows, err := reader.db.QueryContext(ctx, `
		WITH latest_market AS (SELECT MAX(trade_date) AS trade_date FROM daily_bars),
		latest_basic AS (SELECT MAX(trade_date) AS trade_date FROM daily_basic),
		latest_financial AS (SELECT MAX(period_end) AS period_end FROM stock_financial_reports WHERE period = 'annual'),
		latest_valuation AS (SELECT MAX(trade_date) AS trade_date FROM stock_valuation_snapshots)
		SELECT i.code, i.name,
			DATE_FORMAT(b.trade_date, '%Y-%m-%d'), CAST(ROUND(b.close_price, 2) AS CHAR), CAST(b.volume AS CHAR), CAST(ROUND(b.turnover_amount, 2) AS CHAR),
			DATE_FORMAT(db.trade_date, '%Y-%m-%d'), CAST(ROUND(db.market_cap, 2) AS CHAR),
			DATE_FORMAT(tr.metric_date, '%Y-%m-%d'), tr.basis, CAST(tr.metric_value AS CHAR),
			DATE_FORMAT(v.trade_date, '%Y-%m-%d'), CAST(v.pe_ttm AS CHAR), v.pe_ttm_basis, CAST(v.pb AS CHAR), v.pb_basis, CAST(v.ps_ttm AS CHAR), v.ps_ttm_basis,
			DATE_FORMAT(f.period_end, '%Y-%m-%d'), CAST(f.revenue AS CHAR), CAST(f.net_profit AS CHAR), CAST(f.total_assets AS CHAR), CAST(f.total_liabilities AS CHAR), CAST(f.total_equity AS CHAR), CAST(f.operating_cash_flow AS CHAR),
			DATE_FORMAT(roe.metric_date, '%Y-%m-%d'), roe.basis, CAST(roe.metric_value AS CHAR)
		FROM instruments AS i
		CROSS JOIN latest_market AS lm
		LEFT JOIN daily_bars AS b ON b.instrument_code = i.code AND b.trade_date = lm.trade_date
		LEFT JOIN latest_basic AS lb ON TRUE
		LEFT JOIN daily_basic AS db ON db.instrument_code = i.code AND db.trade_date = lb.trade_date
		LEFT JOIN financial_metrics AS tr ON tr.instrument_code = i.code AND tr.metric_date = lm.trade_date AND tr.metric_name = 'turnover_rate'
		LEFT JOIN latest_valuation AS lv ON TRUE
		LEFT JOIN stock_valuation_snapshots AS v ON v.instrument_code = i.code AND v.trade_date = lv.trade_date
		LEFT JOIN latest_financial AS lf ON TRUE
		LEFT JOIN stock_financial_reports AS f ON f.instrument_code = i.code AND f.period = 'annual' AND f.period_end = lf.period_end
		LEFT JOIN financial_metrics AS roe ON roe.instrument_code = i.code AND roe.metric_name = 'roe' AND roe.metric_date = (SELECT MAX(metric_date) FROM financial_metrics WHERE instrument_code = i.code AND metric_name = 'roe')
		WHERE i.status = 'active'
		ORDER BY i.code`)
	if err != nil {
		return nil, fmt.Errorf("read screener candidates: %w", err)
	}
	defer rows.Close()
	candidates := make([]domain.Candidate, 0)
	for rows.Next() {
		candidate, err := scanCandidate(rows)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate screener candidates: %w", err)
	}
	return candidates, nil
}

func scanCandidate(rows *sql.Rows) (domain.Candidate, error) {
	var candidate domain.Candidate
	var marketAsOf, closeValue, volume, turnoverAmount, basicAsOf, marketCap sql.NullString
	var turnoverAsOf, turnoverBasis, turnoverRate sql.NullString
	var valuationAsOf, pe, peBasis, pb, pbBasis, ps, psBasis sql.NullString
	var financialAsOf, revenue, netProfit, totalAssets, totalLiabilities, totalEquity, operatingCashFlow sql.NullString
	var roeAsOf, roeBasis, roe sql.NullString
	if err := rows.Scan(&candidate.Symbol, &candidate.Name, &marketAsOf, &closeValue, &volume, &turnoverAmount, &basicAsOf, &marketCap, &turnoverAsOf, &turnoverBasis, &turnoverRate, &valuationAsOf, &pe, &peBasis, &pb, &pbBasis, &ps, &psBasis, &financialAsOf, &revenue, &netProfit, &totalAssets, &totalLiabilities, &totalEquity, &operatingCashFlow, &roeAsOf, &roeBasis, &roe); err != nil {
		return domain.Candidate{}, fmt.Errorf("scan screener candidate: %w", err)
	}
	candidate.Values = map[string]domain.Observation{
		"market.market_cap":               observation(marketCap, "CNY", "latest_daily_basic", nullString(basicAsOf)),
		"market.turnover_rate":            observation(turnoverRate, "%", nullString(turnoverBasis), nullString(turnoverAsOf)),
		"technical.close":                 observation(closeValue, "CNY", "daily_close", nullString(marketAsOf)),
		"technical.volume":                observation(volume, "股", "daily_bar", nullString(marketAsOf)),
		"technical.turnover_amount":       observation(turnoverAmount, "CNY", "daily_bar", nullString(marketAsOf)),
		"valuation.pe_ttm":                observation(pe, "倍", nullString(peBasis), nullString(valuationAsOf)),
		"valuation.pb":                    observation(pb, "倍", nullString(pbBasis), nullString(valuationAsOf)),
		"valuation.ps_ttm":                observation(ps, "倍", nullString(psBasis), nullString(valuationAsOf)),
		"fundamental.revenue":             observation(revenue, "CNY", "latest_annual_report", nullString(financialAsOf)),
		"fundamental.net_profit":          observation(netProfit, "CNY", "latest_annual_report", nullString(financialAsOf)),
		"fundamental.total_assets":        observation(totalAssets, "CNY", "latest_annual_report", nullString(financialAsOf)),
		"fundamental.total_liabilities":   observation(totalLiabilities, "CNY", "latest_annual_report", nullString(financialAsOf)),
		"fundamental.total_equity":        observation(totalEquity, "CNY", "latest_annual_report", nullString(financialAsOf)),
		"fundamental.operating_cash_flow": observation(operatingCashFlow, "CNY", "latest_annual_report", nullString(financialAsOf)),
		"fundamental.roe_pct":             observation(roe, "%", nullString(roeBasis), nullString(roeAsOf)),
	}
	return candidate, nil
}

func observation(value sql.NullString, unit, basis, asOf string) domain.Observation {
	result := domain.Observation{Unit: unit, Basis: basis, AsOf: asOf}
	if value.Valid {
		result.Value = &value.String
	} else {
		reason := "该字段在当前执行快照中不可用"
		result.UnavailableReason = &reason
	}
	return result
}

func (reader *MySQLReader) readIndustries(ctx context.Context, candidates []domain.Candidate) error {
	rows, err := reader.db.QueryContext(ctx, `
		SELECT sm.instrument_code, GROUP_CONCAT(DISTINCT sc.name ORDER BY sc.name SEPARATOR '、')
		FROM sector_memberships AS sm
		INNER JOIN sector_categories AS sc ON sc.code = sm.sector_code
		INNER JOIN instruments AS i ON i.code = sm.instrument_code
		WHERE i.status = 'active'
		GROUP BY sm.instrument_code`)
	if err != nil {
		return fmt.Errorf("read screener industries: %w", err)
	}
	defer rows.Close()
	bySymbol := make(map[string]*domain.Candidate, len(candidates))
	for index := range candidates {
		bySymbol[candidates[index].Symbol] = &candidates[index]
	}
	for rows.Next() {
		var symbol, industries sql.NullString
		if err := rows.Scan(&symbol, &industries); err != nil {
			return fmt.Errorf("scan screener industries: %w", err)
		}
		candidate := bySymbol[symbol.String]
		if candidate == nil || !industries.Valid || industries.String == "" {
			continue
		}
		candidate.Industries = strings.Split(industries.String, "、")
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate screener industries: %w", err)
	}
	return nil
}

func nullString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

var _ interface {
	ReadSnapshot(context.Context, []string) (domain.ExecutionInput, error)
} = (*MySQLReader)(nil)
