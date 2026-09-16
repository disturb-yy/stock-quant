package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/disturb-yy/stock-quant/internal/stock"
	stockdomain "github.com/disturb-yy/stock-quant/internal/stock/domain"
)

// MySQLFinancialsReader 从 MySQL 读取股票财务报告及 Seed 元数据。
type MySQLFinancialsReader struct {
	db *sql.DB
}

// NewMySQLFinancialsReader 创建股票财务报告 MySQL 读取器。
func NewMySQLFinancialsReader(db *sql.DB) (*MySQLFinancialsReader, error) {
	if db == nil {
		return nil, errors.New("stock financials database connection is required")
	}
	return &MySQLFinancialsReader{db: db}, nil
}

// ReadStockFinancials 读取同一期间口径的全部报告，范围由 Application 层截取。
func (reader *MySQLFinancialsReader) ReadStockFinancials(ctx context.Context, request stock.FinancialsRequest) (stock.FinancialsSnapshot, error) {
	snapshot, err := reader.readFinancialIdentity(ctx, request.Symbol)
	if err != nil {
		return stock.FinancialsSnapshot{}, err
	}
	snapshot.SeedVersion, snapshot.AsOf, err = reader.readFinancialSource(ctx)
	if err != nil {
		return stock.FinancialsSnapshot{}, err
	}
	snapshot.Reports, err = reader.readFinancialReports(ctx, request)
	if err != nil {
		return stock.FinancialsSnapshot{}, err
	}
	return snapshot, nil
}

func (reader *MySQLFinancialsReader) readFinancialIdentity(ctx context.Context, symbol string) (stock.FinancialsSnapshot, error) {
	var snapshot stock.FinancialsSnapshot
	err := reader.db.QueryRowContext(ctx, `SELECT code, name FROM instruments WHERE code = ?`, symbol).Scan(&snapshot.Symbol, &snapshot.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return stock.FinancialsSnapshot{}, stock.ErrFinancialsInstrumentNotFound
	}
	if err != nil {
		return stock.FinancialsSnapshot{}, fmt.Errorf("read stock financial identity: %w", err)
	}
	return snapshot, nil
}

func (reader *MySQLFinancialsReader) readFinancialSource(ctx context.Context) (string, string, error) {
	var seedVersion, asOf string
	err := reader.db.QueryRowContext(ctx, `
		SELECT seed_version, DATE_FORMAT(as_of, '%Y-%m-%d')
		FROM demo_seed_metadata WHERE seed_name = ?`, "fnd-003-demo").Scan(&seedVersion, &asOf)
	if err != nil {
		return "", "", fmt.Errorf("read stock financial source: %w", err)
	}
	return seedVersion, asOf, nil
}

func (reader *MySQLFinancialsReader) readFinancialReports(ctx context.Context, request stock.FinancialsRequest) ([]stockdomain.FinancialReport, error) {
	rows, err := reader.db.QueryContext(ctx, `
		SELECT DATE_FORMAT(period_end, '%Y-%m-%d'), fiscal_year, fiscal_quarter,
		       DATE_FORMAT(published_at, '%Y-%m-%d'),
		       CAST(ROUND(revenue, 2) AS CHAR), CAST(ROUND(gross_profit, 2) AS CHAR),
		       CAST(ROUND(operating_profit, 2) AS CHAR), CAST(ROUND(net_profit, 2) AS CHAR),
		       CAST(ROUND(cash_and_equivalents, 2) AS CHAR), CAST(ROUND(accounts_receivable, 2) AS CHAR),
		       CAST(ROUND(inventory, 2) AS CHAR), CAST(ROUND(current_assets, 2) AS CHAR),
		       CAST(ROUND(current_liabilities, 2) AS CHAR), CAST(ROUND(total_assets, 2) AS CHAR),
		       CAST(ROUND(total_liabilities, 2) AS CHAR), CAST(ROUND(total_equity, 2) AS CHAR),
		       CAST(ROUND(operating_cash_flow, 2) AS CHAR), CAST(ROUND(capital_expenditure, 2) AS CHAR),
		       CAST(ROUND(investing_cash_flow, 2) AS CHAR), CAST(ROUND(financing_cash_flow, 2) AS CHAR),
		       CAST(ROUND(net_cash_change, 2) AS CHAR)
		FROM stock_financial_reports
		WHERE instrument_code = ? AND period = ?
		ORDER BY period_end ASC`, request.Symbol, request.Period)
	if err != nil {
		return nil, fmt.Errorf("read stock financial reports: %w", err)
	}
	defer rows.Close()
	reports := make([]stockdomain.FinancialReport, 0)
	for rows.Next() {
		report, err := scanFinancialReport(rows, request.Symbol, request.Period)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stock financial reports: %w", err)
	}
	return reports, nil
}

func scanFinancialReport(rows *sql.Rows, symbol string, period stockdomain.FinancialPeriod) (stockdomain.FinancialReport, error) {
	var report stockdomain.FinancialReport
	var periodEnd, fiscalQuarter, publishedAt sql.NullString
	var fiscalYear int
	values := make([]sql.NullString, 17)
	err := rows.Scan(&periodEnd, &fiscalYear, &fiscalQuarter, &publishedAt,
		&values[0], &values[1], &values[2], &values[3], &values[4], &values[5], &values[6], &values[7], &values[8], &values[9], &values[10], &values[11], &values[12], &values[13], &values[14], &values[15], &values[16])
	if err != nil {
		return stockdomain.FinancialReport{}, fmt.Errorf("scan stock financial report: %w", err)
	}
	if !periodEnd.Valid || !fiscalQuarter.Valid && period == stockdomain.PeriodQuarterly {
		return stockdomain.FinancialReport{}, errors.New("stock financial report period fields are invalid")
	}
	report = stockdomain.FinancialReport{InstrumentCode: symbol, Period: period, PeriodEnd: periodEnd.String, FiscalYear: fiscalYear, FiscalQuarter: fiscalQuarter.String, PublishedAt: nullableString(publishedAt)}
	report.Income = stockdomain.IncomeStatement{Revenue: nullableString(values[0]), GrossProfit: nullableString(values[1]), OperatingProfit: nullableString(values[2]), NetProfit: nullableString(values[3])}
	report.Balance = stockdomain.BalanceSheet{CashAndEquivalents: nullableString(values[4]), AccountsReceivable: nullableString(values[5]), Inventory: nullableString(values[6]), CurrentAssets: nullableString(values[7]), CurrentLiabilities: nullableString(values[8]), TotalAssets: nullableString(values[9]), TotalLiabilities: nullableString(values[10]), TotalEquity: nullableString(values[11])}
	report.CashFlow = stockdomain.CashFlowStatement{OperatingCashFlow: nullableString(values[12]), CapitalExpenditure: nullableString(values[13]), InvestingCashFlow: nullableString(values[14]), FinancingCashFlow: nullableString(values[15]), NetCashChange: nullableString(values[16])}
	return report, nil
}

var _ stock.FinancialsReader = (*MySQLFinancialsReader)(nil)
