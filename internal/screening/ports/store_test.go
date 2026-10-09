package ports

import (
	"context"
	"reflect"
	"testing"
	"time"

	"stock-quant/internal/screening/domain"
	"stock-quant/internal/shared/types"
)

type fakeScreeningStore struct {
	saved domain.RunMetadata
}

func (store *fakeScreeningStore) CreateOrGet(_ context.Context, run domain.RunMetadata) (domain.RunMetadata, bool, error) {
	store.saved = run
	return run, true, nil
}

func (*fakeScreeningStore) MarkRunning(context.Context, string, time.Time) error { return nil }
func (*fakeScreeningStore) MarkFailed(context.Context, string, string, string, time.Time) error {
	return nil
}
func (*fakeScreeningStore) MarkBlocked(context.Context, string, string, string, time.Time) error {
	return nil
}
func (*fakeScreeningStore) Complete(context.Context, string, domain.ScreeningOutcome, time.Time) error {
	return nil
}
func (store *fakeScreeningStore) Find(context.Context, string) (domain.RunMetadata, error) {
	return store.saved, nil
}
func (*fakeScreeningStore) ListResults(context.Context, string, int, int) ([]domain.ScreeningResult, error) {
	return nil, nil
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
	got, created, err := store.CreateOrGet(context.Background(), want)
	if err != nil || !created {
		t.Fatalf("CreateOrGet() = %#v, %v, %v", got, created, err)
	}
	if !reflect.DeepEqual(store.saved, want) {
		t.Fatalf("SaveRun() stored %#v, want %#v", store.saved, want)
	}
}
