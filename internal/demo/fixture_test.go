package demo

import "testing"

func TestDemoFixtureIsVersionedAndDeterministic(t *testing.T) {
	first := DemoFixture()
	second := DemoFixture()

	if err := first.Validate(); err != nil {
		t.Fatalf("validate fixture: %v", err)
	}
	if first.Version != SeedVersion || first.AsOf != SeedAsOf {
		t.Fatalf("fixture identity = %q/%q, want %q/%q", first.Version, first.AsOf, SeedVersion, SeedAsOf)
	}
	if got, want := first.DataCounts(), (Counts{Instruments: 10, DailyBars: 1210, FinancialMetrics: 16, DailyBasics: 3, IndexSnapshots: 124}); got != want {
		t.Fatalf("fixture counts = %#v, want %#v", got, want)
	}
	if first.DailyBars[120].TradeDate != SeedAsOf || len(first.DailyBars) < 121 {
		t.Fatalf("fixture daily bars do not include latest signal observation: %#v", first.DailyBars[120])
	}
	if first.Instruments[0] != second.Instruments[0] || first.DailyBars[0] != second.DailyBars[0] || first.FinancialMetrics[0] != second.FinancialMetrics[0] || first.IndexSnapshots[0] != second.IndexSnapshots[0] {
		t.Fatal("fixture values are not deterministic")
	}
}
