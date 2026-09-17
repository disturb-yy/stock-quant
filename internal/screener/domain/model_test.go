package domain

import "testing"

func TestNormalizeAndValidateSpec(t *testing.T) {
	tests := []struct {
		name      string
		spec      ScreenerSpec
		wantError bool
	}{
		{
			name: "normalizes canonical request",
			spec: ScreenerSpec{UniverseID: " cn_a_share_active ", Filters: []Filter{{FieldID: " technical.close ", Operator: OperatorGreaterEqual, Value: "10.00"}}, Ranking: Ranking{FieldID: "technical.volume", Direction: "DESC"}, TopN: 10},
		},
		{
			name:      "rejects unknown field",
			spec:      ScreenerSpec{UniverseID: ActiveAShareUniverse, Filters: []Filter{{FieldID: "factor.unknown", Operator: OperatorEqual, Value: "1"}}, Ranking: Ranking{FieldID: "technical.close", Direction: "desc"}, TopN: 1},
			wantError: true,
		},
		{
			name:      "rejects numeric JSON value",
			spec:      ScreenerSpec{UniverseID: ActiveAShareUniverse, Filters: []Filter{{FieldID: "technical.close", Operator: OperatorEqual, Value: 10}}, Ranking: Ranking{FieldID: "technical.close", Direction: "desc"}, TopN: 1},
			wantError: true,
		},
		{
			name:      "rejects top n outside range",
			spec:      ScreenerSpec{UniverseID: ActiveAShareUniverse, Ranking: Ranking{FieldID: "technical.close", Direction: "desc"}, TopN: MaxTopN + 1},
			wantError: true,
		},
		{
			name:      "rejects reversed between",
			spec:      ScreenerSpec{UniverseID: ActiveAShareUniverse, Filters: []Filter{{FieldID: "technical.close", Operator: OperatorBetween, Value: []any{"2", "1"}}}, Ranking: Ranking{FieldID: "technical.close", Direction: "desc"}, TopN: 1},
			wantError: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NormalizeAndValidateSpec(test.spec)
			if test.wantError {
				if err == nil {
					t.Fatal("expected validation error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.UniverseID != ActiveAShareUniverse || got.Ranking.Direction != "desc" {
				t.Fatalf("normalized spec = %#v", got)
			}
		})
	}
}

func TestNormalizeAndValidateSpecUnknownUniverse(t *testing.T) {
	_, err := NormalizeAndValidateSpec(ScreenerSpec{UniverseID: "unknown", Ranking: Ranking{FieldID: "technical.close", Direction: "desc"}, TopN: 1})
	if err != ErrUniverseNotFound {
		t.Fatalf("error = %v, want ErrUniverseNotFound", err)
	}
}

func TestExecuteUsesANDStableRankingAndTopN(t *testing.T) {
	value := func(value string) *string { return &value }
	input := ExecutionInput{
		Universe: Universe{ID: ActiveAShareUniverse, Name: ActiveAShareName},
		Eligible: []Candidate{
			{Symbol: "B.SH", Name: "乙", Values: map[string]Observation{"technical.close": {Value: value("10"), Basis: "daily_close", AsOf: "2024-06-28"}, "market.market_cap": {Value: value("100"), Basis: "latest_daily_basic", AsOf: "2024-06-28"}}},
			{Symbol: "A.SH", Name: "甲", Values: map[string]Observation{"technical.close": {Value: value("10"), Basis: "daily_close", AsOf: "2024-06-28"}, "market.market_cap": {Value: value("200"), Basis: "latest_daily_basic", AsOf: "2024-06-28"}}},
			{Symbol: "C.SH", Name: "丙", Values: map[string]Observation{"technical.close": {Value: value("8"), Basis: "daily_close", AsOf: "2024-06-28"}, "market.market_cap": {Value: value("300"), Basis: "latest_daily_basic", AsOf: "2024-06-28"}}},
		},
		Snapshot: Snapshot{AsOf: "2024-06-28", FieldAsOf: map[string]string{"technical.close": "2024-06-28"}, DefinitionVersions: map[string]string{}},
		Source:   Source{Mode: "demo", Provider: "mysql-demo-fixture", SeedVersion: "fnd-003-demo-v8", AsOf: "2024-06-28"},
	}
	result, err := Execute(ScreenerSpec{UniverseID: ActiveAShareUniverse, Filters: []Filter{{FieldID: "technical.close", Operator: OperatorGreaterEqual, Value: "10"}, {FieldID: "market.market_cap", Operator: OperatorGreaterEqual, Value: "150"}}, Ranking: Ranking{FieldID: "technical.close", Direction: "desc"}, TopN: 1}, input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.MatchedCount != 1 || result.ReturnedCount != 1 || result.Results[0].Symbol != "A.SH" || result.Results[0].Rank != 1 {
		t.Fatalf("result = %#v", result)
	}
}

func TestExecuteExcludesUnavailableRankingValue(t *testing.T) {
	input := ExecutionInput{Universe: Universe{ID: ActiveAShareUniverse, Name: ActiveAShareName}, Eligible: []Candidate{{Symbol: "A.SH", Values: map[string]Observation{"technical.close": {}}}}, Source: Source{Mode: "demo", Provider: "seed", SeedVersion: "v1", AsOf: "2024-01-01"}}
	result, err := Execute(ScreenerSpec{UniverseID: ActiveAShareUniverse, Ranking: Ranking{FieldID: "technical.close", Direction: "desc"}, TopN: 1}, input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.MatchedCount != 0 || len(result.Results) != 0 {
		t.Fatalf("result = %#v, want empty sortable result", result)
	}
}

func TestExecuteSupportsAllCanonicalOperators(t *testing.T) {
	value := "10"
	base := ExecutionInput{Universe: Universe{ID: ActiveAShareUniverse, Name: ActiveAShareName}, Eligible: []Candidate{{Symbol: "A.SH", Values: map[string]Observation{"technical.close": {Value: &value}}}}}
	tests := []struct {
		name     string
		operator Operator
		value    any
		matched  bool
	}{
		{name: "eq", operator: OperatorEqual, value: "10", matched: true},
		{name: "neq", operator: OperatorNotEqual, value: "9", matched: true},
		{name: "gt", operator: OperatorGreater, value: "9", matched: true},
		{name: "gte", operator: OperatorGreaterEqual, value: "10", matched: true},
		{name: "lt", operator: OperatorLess, value: "11", matched: true},
		{name: "lte", operator: OperatorLessEqual, value: "10", matched: true},
		{name: "between", operator: OperatorBetween, value: []any{"9", "10"}, matched: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spec := ScreenerSpec{UniverseID: ActiveAShareUniverse, Filters: []Filter{{FieldID: "technical.close", Operator: test.operator, Value: test.value}}, Ranking: Ranking{FieldID: "technical.close", Direction: "desc"}, TopN: 1}
			result, err := Execute(spec, base)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if (result.MatchedCount == 1) != test.matched {
				t.Fatalf("matched_count = %d, want matched=%t", result.MatchedCount, test.matched)
			}
		})
	}
}
