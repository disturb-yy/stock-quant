package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/disturb-yy/stock-quant/internal/pool/domain"
)

const (
	industrySummaryProvenance = "sector_memberships"
	metricSummaryProvenance   = "financial_metrics"
)

// Summary 在一个只读事务内读取 Pool 元数据、成员聚合和指标摘要。
func (store *MySQLStockPoolStore) Summary(ctx context.Context, id int64) (domain.StockPoolSummary, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.StockPoolSummary{}, fmt.Errorf("begin stock pool summary: %w", err)
	}
	defer rollbackTransaction(tx)

	pool, source, err := readSummaryPool(ctx, tx, id)
	if err != nil {
		return domain.StockPoolSummary{}, err
	}
	summary := newStockPoolSummary(pool, source)
	if pool.MemberCount == 0 {
		markSummaryEmpty(&summary)
	} else {
		if err := fillIndustrySummary(ctx, tx, id, &summary); err != nil {
			return domain.StockPoolSummary{}, err
		}
		if err := fillMetricSummaries(ctx, tx, id, pool.MemberCount, &summary); err != nil {
			return domain.StockPoolSummary{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.StockPoolSummary{}, fmt.Errorf("commit stock pool summary: %w", err)
	}
	return summary, nil
}

func readSummaryPool(ctx context.Context, tx *sql.Tx, id int64) (domain.StockPool, domain.StockPoolSourceSummary, error) {
	pool, err := readStockPool(ctx, tx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.StockPool{}, domain.StockPoolSourceSummary{}, domain.ErrStockPoolNotFound
	}
	if err != nil {
		return domain.StockPool{}, domain.StockPoolSourceSummary{}, fmt.Errorf("read stock pool summary metadata: %w", err)
	}
	if pool.Source != domain.SourceManual {
		return domain.StockPool{}, domain.StockPoolSourceSummary{}, domain.ErrStockPoolSourceMetadataUnavailable
	}
	reference, err := readPoolSeedKey(ctx, tx, id)
	if err != nil {
		return domain.StockPool{}, domain.StockPoolSourceSummary{}, err
	}
	return pool, domain.StockPoolSourceSummary{
		Type: pool.Source, Reference: reference, CreatedAt: pool.CreatedAt,
	}, nil
}

func readPoolSeedKey(ctx context.Context, tx *sql.Tx, id int64) (*string, error) {
	var seedKey sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT seed_key FROM t_stock_pool WHERE id = ?`, id).Scan(&seedKey); err != nil {
		return nil, fmt.Errorf("read stock pool source reference: %w", err)
	}
	if !seedKey.Valid {
		return nil, nil
	}
	return &seedKey.String, nil
}

func newStockPoolSummary(pool domain.StockPool, source domain.StockPoolSourceSummary) domain.StockPoolSummary {
	return domain.StockPoolSummary{
		ID: pool.ID, Name: pool.Name, Description: pool.Description, Source: source,
		MemberCount: pool.MemberCount, CreatedAt: pool.CreatedAt, UpdatedAt: pool.UpdatedAt,
	}
}

func markSummaryEmpty(summary *domain.StockPoolSummary) {
	reason := "股票池无成员"
	summary.Industry = domain.StockPoolIndustrySummary{Availability: domain.SummaryEmpty, UnavailableReason: &reason}
	summary.PE = emptyMetricSummary(reason)
	summary.ROE = emptyMetricSummary(reason)
}

func emptyMetricSummary(reason string) domain.StockPoolMetricSummary {
	return domain.StockPoolMetricSummary{Availability: domain.SummaryEmpty, UnavailableReason: stringPointer(reason)}
}

func fillIndustrySummary(ctx context.Context, tx *sql.Tx, id int64, summary *domain.StockPoolSummary) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT sectors.code, sectors.name, COUNT(DISTINCT member.symbol)
		FROM t_stock_pool_member AS member
		INNER JOIN sector_memberships AS memberships ON memberships.instrument_code = member.symbol
		INNER JOIN sector_categories AS sectors ON sectors.code = memberships.sector_code
		WHERE member.pool_id = ?
		GROUP BY sectors.code, sectors.name
		ORDER BY sectors.name ASC, sectors.code ASC`, id)
	if err != nil {
		return fmt.Errorf("read stock pool industry summary: %w", err)
	}
	defer rows.Close()
	distribution := make([]domain.StockPoolIndustryBucket, 0)
	for rows.Next() {
		var bucket domain.StockPoolIndustryBucket
		if err := rows.Scan(&bucket.Code, &bucket.Name, &bucket.MemberCount); err != nil {
			return fmt.Errorf("scan stock pool industry summary: %w", err)
		}
		distribution = append(distribution, bucket)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate stock pool industry summary: %w", err)
	}
	if len(distribution) == 0 {
		reason := "成员没有可用的行业归属"
		summary.Industry = domain.StockPoolIndustrySummary{Availability: domain.SummaryUnavailable, UnavailableReason: &reason}
		return nil
	}
	provenance := industrySummaryProvenance
	summary.Industry = domain.StockPoolIndustrySummary{
		Availability: domain.SummaryAvailable, Distribution: distribution, Provenance: &provenance,
	}
	return nil
}

func fillMetricSummaries(ctx context.Context, tx *sql.Tx, id, memberCount int64, summary *domain.StockPoolSummary) error {
	rows, err := tx.QueryContext(ctx, `
		WITH latest_metric AS (
			SELECT metrics.instrument_code, metrics.metric_name, metrics.metric_date, metrics.basis, metrics.metric_value
			FROM financial_metrics AS metrics
			INNER JOIN t_stock_pool_member AS member ON member.symbol = metrics.instrument_code
			WHERE member.pool_id = ?
			  AND metrics.metric_name IN ('pe_ttm', 'roe')
			  AND metrics.metric_date = (
				  SELECT MAX(latest.metric_date)
				  FROM financial_metrics AS latest
				  WHERE latest.instrument_code = metrics.instrument_code
				    AND latest.metric_name = metrics.metric_name
			  )
		)
		SELECT metric_name, CAST(ROUND(AVG(metric_value), 2) AS CHAR), COUNT(metric_value),
		       DATE_FORMAT(MAX(metric_date), '%Y-%m-%d'),
		       CASE WHEN COUNT(DISTINCT basis) = 1 THEN MAX(basis) END
		FROM latest_metric
		GROUP BY metric_name`, id)
	if err != nil {
		return fmt.Errorf("read stock pool metric summary: %w", err)
	}
	defer rows.Close()
	metrics := make(map[string]domain.StockPoolMetricSummary, 2)
	for rows.Next() {
		metric, name, err := scanMetricSummary(rows, memberCount)
		if err != nil {
			return err
		}
		metrics[name] = metric
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate stock pool metric summary: %w", err)
	}
	summary.PE = metricOrUnavailable(metrics["pe_ttm"], "PE 没有可用数据")
	summary.ROE = metricOrUnavailable(metrics["roe"], "ROE 没有可用数据")
	return nil
}

func scanMetricSummary(rows *sql.Rows, memberCount int64) (domain.StockPoolMetricSummary, string, error) {
	var name, value, asOf sql.NullString
	var sampleSize int64
	var basis sql.NullString
	if err := rows.Scan(&name, &value, &sampleSize, &asOf, &basis); err != nil {
		return domain.StockPoolMetricSummary{}, "", fmt.Errorf("scan stock pool metric summary: %w", err)
	}
	if !value.Valid || !asOf.Valid || sampleSize == 0 {
		return domain.StockPoolMetricSummary{Availability: domain.SummaryUnavailable}, name.String, nil
	}
	provenance := metricSummaryProvenance
	metric := domain.StockPoolMetricSummary{
		Availability: domain.SummaryAvailable, Value: &value.String, SampleSize: sampleSize,
		AsOf: &asOf.String, Provenance: &provenance,
	}
	if basis.Valid {
		metric.Basis = &basis.String
	}
	if sampleSize < memberCount {
		reason := "部分成员没有可用数据"
		metric.UnavailableReason = &reason
	}
	return metric, name.String, nil
}

func metricOrUnavailable(metric domain.StockPoolMetricSummary, reason string) domain.StockPoolMetricSummary {
	if metric.Availability == domain.SummaryUnavailable && metric.UnavailableReason == nil {
		reasonValue := reason
		metric.UnavailableReason = &reasonValue
	}
	if metric.Availability != "" {
		return metric
	}
	reasonValue := reason
	return domain.StockPoolMetricSummary{Availability: domain.SummaryUnavailable, UnavailableReason: &reasonValue}
}

func stringPointer(value string) *string {
	return &value
}
