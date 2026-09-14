package migration

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type fakeMigration struct {
	name  string
	steps *[]string
	err   error
}

func (migration fakeMigration) Name() string {
	return migration.name
}

func (migration fakeMigration) Up(context.Context) error {
	if migration.steps != nil {
		*migration.steps = append(*migration.steps, migration.name)
	}
	return migration.err
}

func TestNewRunnerAllowsEmptyCollection(t *testing.T) {
	runner, err := NewRunner()
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}
	if runner.Count() != 0 {
		t.Fatalf("Count() = %d, want 0", runner.Count())
	}
	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestNewRunnerPreservesRegistrationOrder(t *testing.T) {
	steps := []string{}
	runner, err := NewRunner(
		fakeMigration{name: "0002_second", steps: &steps},
		fakeMigration{name: "0001_first", steps: &steps},
	)
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}
	if runner.Count() != 2 {
		t.Fatalf("Count() = %d, want 2", runner.Count())
	}

	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if want := []string{"0002_second", "0001_first"}; !reflect.DeepEqual(steps, want) {
		t.Fatalf("migration order = %#v, want %#v", steps, want)
	}
}

func TestNewRunnerRejectsInvalidDefinitions(t *testing.T) {
	tests := []struct {
		name       string
		migrations []Migration
	}{
		{
			name:       "nil migration",
			migrations: []Migration{nil},
		},
		{
			name:       "missing name",
			migrations: []Migration{fakeMigration{}},
		},
		{
			name: "duplicate name",
			migrations: []Migration{
				fakeMigration{name: "0001_same"},
				fakeMigration{name: "0001_same"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewRunner(test.migrations...); err == nil {
				t.Fatal("NewRunner() error = nil, want error")
			}
		})
	}
}

func TestRunnerRunWrapsMigrationError(t *testing.T) {
	wantErr := errors.New("schema unavailable")
	runner, err := NewRunner(fakeMigration{name: "0001_initial", err: wantErr})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	if err := runner.Run(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want wrapped %v", err, wantErr)
	}
}

func TestRunnerStopsWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	steps := []string{}
	runner, err := NewRunner(fakeMigration{name: "0001_initial", steps: &steps})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	if err := runner.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context.Canceled", err)
	}
	if len(steps) != 0 {
		t.Fatalf("migration steps = %#v, want no steps", steps)
	}
}
