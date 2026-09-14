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
	if got, want := first.DataCounts(), (Counts{Instruments: 3, DailyBars: 6, FinancialMetrics: 6, IndexSnapshots: 4}); got != want {
		t.Fatalf("fixture counts = %#v, want %#v", got, want)
	}
	if first.Instruments[0] != second.Instruments[0] || first.DailyBars[0] != second.DailyBars[0] || first.FinancialMetrics[0] != second.FinancialMetrics[0] || first.IndexSnapshots[0] != second.IndexSnapshots[0] {
		t.Fatal("fixture values are not deterministic")
	}
}
