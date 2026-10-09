package ports

import (
	"context"
	"testing"

	"stock-quant/internal/screening/domain"
	"stock-quant/internal/shared/types"
)

type fakeScreeningStore struct {
	saved domain.RunMetadata
}

func (store *fakeScreeningStore) SaveRun(_ context.Context, run domain.RunMetadata) error {
	store.saved = run
	return nil
}

var _ ScreeningStore = (*fakeScreeningStore)(nil)

func TestScreeningStoreFakeReceivesRunIdentity(t *testing.T) {
	asOf, err := types.ParseTradingDate("2026-10-08")
	if err != nil {
		t.Fatal(err)
	}
	want := domain.RunMetadata{
		RunID:           "run-001",
		AsOf:            asOf,
		StrategyID:      "momentum_v1",
		StrategyVersion: "1.0.0",
		SnapshotHash:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	store := &fakeScreeningStore{}
	if err := store.SaveRun(context.Background(), want); err != nil {
		t.Fatalf("SaveRun() error = %v", err)
	}
	if store.saved != want {
		t.Fatalf("SaveRun() stored %#v, want %#v", store.saved, want)
	}
}
