package demo

import (
	"context"
	"errors"
	"testing"
)

type fakeStatusStore struct {
	snapshot StoreSnapshot
	err      error
}

func (store fakeStatusStore) ReadDemoSnapshot(context.Context) (StoreSnapshot, error) {
	return store.snapshot, store.err
}

type fakeSeedStore struct {
	calls   int
	fixture Fixture
}

func (store *fakeSeedStore) SeedDemo(_ context.Context, fixture Fixture) error {
	store.calls++
	store.fixture = fixture
	return nil
}

func TestServiceDemoStatusModes(t *testing.T) {
	fixture := DemoFixture()
	asOf := SeedAsOf
	tests := []struct {
		name              string
		requestedProvider string
		snapshot          StoreSnapshot
		wantMode          string
		wantProvider      string
	}{
		{
			name:              "demo fixture",
			requestedProvider: "demo",
			snapshot:          StoreSnapshot{SeedVersion: SeedVersion, AsOf: &asOf, Counts: fixture.DataCounts()},
			wantMode:          "demo",
			wantProvider:      "mysql-demo-fixture",
		},
		{
			name:              "real provider falls back",
			requestedProvider: "real",
			snapshot:          StoreSnapshot{SeedVersion: SeedVersion, AsOf: &asOf, Counts: fixture.DataCounts()},
			wantMode:          "fallback",
			wantProvider:      "local-fixture-fallback",
		},
		{
			name:              "empty database falls back",
			requestedProvider: "demo",
			snapshot:          StoreSnapshot{Counts: Counts{}, Samples: []SampleStock{}},
			wantMode:          "fallback",
			wantProvider:      "local-fixture-fallback",
		},
		{
			name:              "wrong fixture date falls back",
			requestedProvider: "demo",
			snapshot:          StoreSnapshot{SeedVersion: SeedVersion, AsOf: stringPointer("2024-06-27"), Counts: fixture.DataCounts()},
			wantMode:          "fallback",
			wantProvider:      "local-fixture-fallback",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, err := NewService(fakeStatusStore{snapshot: test.snapshot}, test.requestedProvider)
			if err != nil {
				t.Fatalf("NewService() error = %v", err)
			}
			status, err := service.DemoStatus(context.Background())
			if err != nil {
				t.Fatalf("DemoStatus() error = %v", err)
			}
			if string(status.Mode) != test.wantMode || status.Provider != test.wantProvider {
				t.Fatalf("mode/provider = %q/%q, want %q/%q", status.Mode, status.Provider, test.wantMode, test.wantProvider)
			}
		})
	}
}

func TestSeederValidatesAndDelegatesFixture(t *testing.T) {
	store := &fakeSeedStore{}
	seeder, err := NewSeeder(store)
	if err != nil {
		t.Fatalf("NewSeeder() error = %v", err)
	}
	fixture := DemoFixture()
	if err := seeder.Seed(context.Background(), fixture); err != nil {
		t.Fatalf("Seed() error = %v", err)
	}
	if store.calls != 1 {
		t.Fatalf("SeedDemo calls = %d, want 1", store.calls)
	}
	if got, want := store.fixture.DataCounts(), fixture.DataCounts(); got != want {
		t.Fatalf("stored counts = %#v, want %#v", got, want)
	}
}

func TestServiceWrapsStoreError(t *testing.T) {
	wantErr := errors.New("database unavailable")
	service, err := NewService(fakeStatusStore{err: wantErr}, "demo")
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if _, err := service.DemoStatus(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("DemoStatus() error = %v, want wrapped %v", err, wantErr)
	}
}

func stringPointer(value string) *string {
	return &value
}
