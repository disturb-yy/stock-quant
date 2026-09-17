package demo

import stockdomain "github.com/disturb-yy/stock-quant/internal/stock/domain"

// valuationFixture 返回可复核的历史估值和同日同行 demo 样本。
func valuationFixture() []stockdomain.ValuationObservation {
	return []stockdomain.ValuationObservation{
		newValuationObservation("000001.SZ", "2020-06-26", "4.20", "0.50", "1.00"),
		newValuationObservation("000001.SZ", "2021-06-28", "5.10", "0.55", "1.10"),
		newValuationObservation("000001.SZ", "2022-06-28", "6.20", "0.60", "1.20"),
		newValuationObservation("000001.SZ", "2023-06-28", "5.82", "0.48", "1.30"),
		newValuationObservation("000001.SZ", "2024-06-28", "7.40", "0.52", "1.40"),
		newValuationObservation("601166.SH", "2024-06-28", "4.00", "0.60", "1.00"),
		newValuationObservation("601398.SH", "2024-06-28", "6.00", "0.70", "1.20"),
		newValuationObservation("600036.SH", "2024-06-28", "8.00", "0.80", "1.40"),
		newValuationObservation("300750.SZ", "2024-06-28", "-1.00", "3.92", "0.00"),
		newValuationObservation("600519.SH", "2024-06-28", "28.63", "8.41", "0.00"),
		newValuationObservation("000858.SZ", "2024-06-28", "12.50", "", "2.10"),
		newValuationObservation("002594.SZ", "2024-06-28", "18.20", "2.30", "1.80"),
	}
}

func newValuationObservation(symbol, asOf, pe, pb, ps string) stockdomain.ValuationObservation {
	return stockdomain.ValuationObservation{
		InstrumentCode: symbol,
		AsOf:           asOf,
		PETTM:          newValuationMetric(asOf, pe, "ttm"),
		PB:             newValuationMetric(asOf, pb, "latest_daily_basic"),
		PSTTM:          newValuationMetric(asOf, ps, "ttm"),
	}
}

func newValuationMetric(asOf, value, basis string) stockdomain.ValuationMetricObservation {
	metric := stockdomain.ValuationMetricObservation{AsOf: asOf}
	if value != "" {
		metric.Value = &value
		metric.Basis = basis
	}
	return metric
}
