package domain

import "testing"

func TestAnalyzeValuationMetricUsesInclusiveRankAndPositions(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		history  []int
		wantRank string
		wantPos  ValuationPosition
	}{
		{name: "low boundary", current: "3", history: []int{1, 2, 4, 5, 6, 7, 8, 9, 10}, wantRank: "30.00", wantPos: PositionMiddle},
		{name: "middle boundary", current: "7", history: []int{1, 2, 3, 4, 5, 6, 8, 9, 10}, wantRank: "70.00", wantPos: PositionMiddle},
		{name: "high", current: "8", history: []int{1, 2, 3, 4, 5, 6, 7, 9, 10}, wantRank: "80.00", wantPos: PositionHigh},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			current := test.current
			input := make([]ValuationMetricObservation, 0, len(test.history)+1)
			for index, historyValue := range append(test.history, 0) {
				value := current
				if index < len(test.history) {
					value = formatTestValue(historyValue)
				}
				input = append(input, ValuationMetricObservation{AsOf: dateForTest(index + 1), Value: &value, Basis: "ttm"})
			}
			result, err := AnalyzeValuationMetric(input, "2020-01-01", "2024-12-31")
			if err != nil {
				t.Fatalf("AnalyzeValuationMetric() error = %v", err)
			}
			if result.Percentile.Value == nil || *result.Percentile.Value != test.wantRank || result.Position == nil || *result.Position != test.wantPos {
				t.Fatalf("percentile/position = %#v/%v, want %s/%s", result.Percentile, result.Position, test.wantRank, test.wantPos)
			}
			if result.Percentile.Method != "inclusive_rank" || result.Percentile.SampleSize != 10 {
				t.Fatalf("percentile method/sample = %#v, want inclusive_rank/10", result.Percentile)
			}
		})
	}
}

func TestAnalyzeValuationMetricExcludesNonPositiveHistoryAndCurrent(t *testing.T) {
	positive := "5"
	zero := "0"
	negative := "-2"
	result, err := AnalyzeValuationMetric([]ValuationMetricObservation{
		{AsOf: "2024-01-01", Value: &positive, Basis: "ttm"},
		{AsOf: "2024-02-01", Value: &zero, Basis: "ttm"},
		{AsOf: "2024-03-01", Value: &negative, Basis: "ttm"},
	}, "2024-01-01", "2024-03-01")
	if err != nil {
		t.Fatalf("AnalyzeValuationMetric() error = %v", err)
	}
	if result.Current.Value == nil || *result.Current.Value != "-2.00" || result.Current.AsOf == nil || *result.Current.AsOf != "2024-03-01" {
		t.Fatalf("current = %#v, want negative current with date", result.Current)
	}
	if len(result.History) != 1 || result.History[0].Value != "5.00" || result.Percentile.Value != nil || result.Position != nil {
		t.Fatalf("derived result = %#v, want one positive history and null derived values", result)
	}
}

func TestCalculateIndustryMedianRequiresSameDateTwoPeersAndPositiveTarget(t *testing.T) {
	target := "10"
	values := []ValuationPeerObservation{
		{InstrumentCode: "000001.SZ", AsOf: "2024-06-28", Value: &target, Basis: "ttm"},
		{InstrumentCode: "601166.SH", AsOf: "2024-06-28", Value: stringPointer("4"), Basis: "ttm"},
		{InstrumentCode: "601398.SH", AsOf: "2024-06-28", Value: stringPointer("8"), Basis: "ttm"},
		{InstrumentCode: "600036.SH", AsOf: "2024-06-27", Value: stringPointer("100"), Basis: "ttm"},
		{InstrumentCode: "600519.SH", AsOf: "2024-06-28", Value: stringPointer("0"), Basis: "ttm"},
	}
	result, err := CalculateIndustryMedian(values, "000001.SZ", "2024-06-28", "ttm", &target)
	if err != nil {
		t.Fatalf("CalculateIndustryMedian() error = %v", err)
	}
	if result.SampleSize != 2 || result.Value == nil || *result.Value != "6.00" {
		t.Fatalf("median = %#v, want 6.00 from two same-day peers", result)
	}

	negativeTarget := "-1"
	result, err = CalculateIndustryMedian(values, "000001.SZ", "2024-06-28", "ttm", &negativeTarget)
	if err != nil {
		t.Fatalf("negative target median error = %v", err)
	}
	if result.Value != nil || result.SampleSize != 2 {
		t.Fatalf("negative target median = %#v, want null with actual same-day sample count", result)
	}
}

func dateForTest(day int) string {
	return "2024-01-" + formatTestValue(day)
}

func formatTestValue(value int) string {
	if value < 10 {
		return "0" + string(rune('0'+value))
	}
	return string(rune('0'+value/10)) + string(rune('0'+value%10))
}

func stringPointer(value string) *string {
	return &value
}
