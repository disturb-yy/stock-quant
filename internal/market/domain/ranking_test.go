package domain

import "testing"

func TestRankObservationsSupportsAllMetricsAndStableTies(t *testing.T) {
	observations := []RankingObservation{
		{InstrumentCode: "600519.SH", InstrumentName: "贵州茅台", CurrentClose: "100", PreviousClose: "100", TurnoverAmount: "300", TurnoverRate: "1.20"},
		{InstrumentCode: "000001.SZ", InstrumentName: "平安银行", CurrentClose: "110", PreviousClose: "100", TurnoverAmount: "300", TurnoverRate: "2.40"},
		{InstrumentCode: "300750.SZ", InstrumentName: "宁德时代", CurrentClose: "90", PreviousClose: "100", TurnoverAmount: "500", TurnoverRate: "2.40"},
	}

	tests := []struct {
		name     string
		metric   RankingMetric
		wantCode []string
		want     string
	}{
		{name: "gain", metric: RankingMetricGain, wantCode: []string{"000001.SZ", "600519.SH", "300750.SZ"}, want: "10.00"},
		{name: "loss", metric: RankingMetricLoss, wantCode: []string{"300750.SZ", "600519.SH", "000001.SZ"}, want: "-10.00"},
		{name: "turnover amount", metric: RankingMetricTurnoverAmount, wantCode: []string{"300750.SZ", "000001.SZ", "600519.SH"}, want: "500"},
		{name: "turnover rate", metric: RankingMetricTurnoverRate, wantCode: []string{"000001.SZ", "300750.SZ", "600519.SH"}, want: "2.40"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := RankObservations(test.metric, observations)
			if err != nil {
				t.Fatalf("RankObservations() error = %v", err)
			}
			if len(got) != len(test.wantCode) {
				t.Fatalf("result count = %d, want %d", len(got), len(test.wantCode))
			}
			for index, code := range test.wantCode {
				if got[index].InstrumentCode != code {
					t.Fatalf("rank %d code = %q, want %q", index+1, got[index].InstrumentCode, code)
				}
				if index == 0 && got[index].Value != test.want {
					t.Fatalf("rank %d value = %q, want %q", index+1, got[index].Value, test.want)
				}
				if got[index].Rank != index+1 {
					t.Fatalf("rank %d result rank = %d, want %d", index+1, got[index].Rank, index+1)
				}
			}
		})
	}
}

func TestRankObservationsRejectsInvalidMetric(t *testing.T) {
	_, err := RankObservations(RankingMetric("unknown"), []RankingObservation{{
		InstrumentCode: "000001.SZ", InstrumentName: "平安银行", CurrentClose: "10", PreviousClose: "9",
		TurnoverAmount: "100", TurnoverRate: "1.00",
	}})
	if err == nil {
		t.Fatal("RankObservations() error = nil, want invalid metric error")
	}
}
