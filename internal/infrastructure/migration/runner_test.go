package migration

import (
	"context"
	"testing"

	"gorm.io/gorm"
)

func TestRunRequiresDatabase(t *testing.T) {
	err := Run(context.Background(), nil, nil, DirectionUp)
	if err == nil || err.Error() != "migration database is nil" {
		t.Fatalf("error = %v, want migration database is nil", err)
	}
}

func TestRunRequiresMigrationFilesystem(t *testing.T) {
	err := Run(context.Background(), &gorm.DB{}, nil, DirectionUp)
	if err == nil || err.Error() != "migration filesystem is nil" {
		t.Fatalf("error = %v, want migration filesystem is nil", err)
	}
}
