package domain

import (
	"strings"
	"testing"

	"stock-quant/internal/shared/types"
)

func TestCanonicalRunKeyIncludesImmutableInputs(t *testing.T) {
	date, err := types.ParseTradingDate("2026-06-04")
	if err != nil {
		t.Fatal(err)
	}
	run := Run{StrategyID: "s1", StrategyVersion: "1", StartDate: date, EndDate: date, ConfigHash: strings.Repeat("a", 64), SnapshotHash: strings.Repeat("b", 64), Mode: "research_only"}
	want := CanonicalRunKey(run)
	if len(want) != 64 {
		t.Fatalf("key length = %d, want SHA-256 hex", len(want))
	}
	if CanonicalRunKey(run) != want {
		t.Fatal("same immutable inputs produced different keys")
	}
	variants := []Run{run, run, run, run, run, run, run}
	variants[0].StrategyID = "s2"
	variants[1].StrategyVersion = "2"
	variants[2].StartDate, _ = types.ParseTradingDate("2026-06-03")
	variants[3].EndDate, _ = types.ParseTradingDate("2026-06-05")
	variants[4].ConfigHash = strings.Repeat("c", 64)
	variants[5].SnapshotHash = strings.Repeat("d", 64)
	variants[6].Mode = "production"
	for i, variant := range variants {
		if CanonicalRunKey(variant) == want {
			t.Errorf("input variant %d did not change run key", i)
		}
	}
}
