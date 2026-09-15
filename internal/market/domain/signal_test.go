package domain

import (
	"math/big"
	"testing"
)

func TestScanSignals(t *testing.T) {
	tests := []struct {
		name       string
		signalType SignalType
		params     SignalRuleParameters
		series     []SignalSeries
		wantCodes  []string
	}{
		{
			name:       "volume surge at threshold",
			signalType: SignalVolumeSurge,
			params:     SignalRuleParameters{Window: 20, Multiple: big.NewRat(3, 2)},
			series:     []SignalSeries{testSeries("A", "甲", 100, 150, 150, 100)},
			wantCodes:  []string{"A"},
		},
		{
			name:       "breakout uses historical high",
			signalType: SignalBreakout,
			params:     SignalRuleParameters{Window: 20},
			series:     []SignalSeries{testSeries("A", "甲", 102, 102, 100, 100)},
			wantCodes:  []string{"A"},
		},
		{
			name:       "new high uses historical close",
			signalType: SignalNewHigh,
			params:     SignalRuleParameters{Window: 20},
			series:     []SignalSeries{testSeries("A", "甲", 101, 101, 100, 100)},
			wantCodes:  []string{"A"},
		},
		{
			name:       "strong selects top percent",
			signalType: SignalStrong,
			params:     SignalRuleParameters{Window: 20, TopPercent: 20},
			series: []SignalSeries{
				testSeries("A", "甲", 100, 101, 100, 100),
				testSeries("B", "乙", 130, 130, 100, 100),
				testSeries("C", "丙", 110, 110, 100, 100),
				testSeries("D", "丁", 105, 105, 100, 100),
			},
			wantCodes: []string{"B"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matches, err := ScanSignals(test.signalType, test.series, test.params)
			if err != nil {
				t.Fatalf("ScanSignals() error = %v", err)
			}
			if len(matches) != len(test.wantCodes) {
				t.Fatalf("match count = %d, want %d", len(matches), len(test.wantCodes))
			}
			for index, code := range test.wantCodes {
				if matches[index].InstrumentCode != code {
					t.Fatalf("match[%d] code = %q, want %q", index, matches[index].InstrumentCode, code)
				}
			}
		})
	}
}

func TestScanSignalsRejectsInsufficientHistory(t *testing.T) {
	series := testSeries("A", "甲", 100, 101, 100, 19)
	series.Bars = series.Bars[:20]
	_, err := ScanSignals(SignalNewHigh, []SignalSeries{series}, SignalRuleParameters{Window: 20})
	if err == nil {
		t.Fatal("ScanSignals() error = nil, want insufficient history")
	}
}

func TestValidateSignalParametersRejectsIrrelevantParameters(t *testing.T) {
	err := ValidateSignalParameters(SignalBreakout, SignalRuleParameters{Window: 20, Multiple: big.NewRat(2, 1)})
	if err == nil {
		t.Fatal("ValidateSignalParameters() error = nil, want irrelevant parameter error")
	}
}

func testSeries(code, name string, currentClose, currentHigh float64, volume int64, previousVolume int64) SignalSeries {
	bars := make([]DailyBar, 0, 21)
	for index := 0; index < 20; index++ {
		bars = append(bars, DailyBar{
			InstrumentCode: code,
			TradeDate:      testDate(index),
			Open:           "99",
			High:           "100",
			Low:            "98",
			Close:          "100",
			Volume:         previousVolume,
			TurnoverAmount: "1",
		})
	}
	bars = append(bars, DailyBar{
		InstrumentCode: code,
		TradeDate:      testDate(20),
		Open:           "100",
		High:           formatTestDecimal(currentHigh),
		Low:            "99",
		Close:          formatTestDecimal(currentClose),
		Volume:         volume,
		TurnoverAmount: "1",
	})
	return SignalSeries{InstrumentCode: code, InstrumentName: name, Bars: bars}
}

func testDate(index int) string {
	return "2024-01-" + formatTestDay(index+1)
}

func formatTestDay(day int) string {
	if day < 10 {
		return "0" + string(rune('0'+day))
	}
	return string(rune('0'+day/10)) + string(rune('0'+day%10))
}

func formatTestDecimal(value float64) string {
	return new(big.Float).SetFloat64(value).Text('f', -1)
}
