package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

func TestEmbeddedMigrationsContainOnlyFEAT001Tables(t *testing.T) {
	entries, err := fs.ReadDir(FS, ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("embedded migration count = %d, want 2", len(entries))
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "" {
			t.Fatalf("unexpected migration entry: %#v", entry)
		}
	}
	if _, err := fs.ReadFile(FS, "000001_create_data_sync_tables.up.sql"); err != nil {
		t.Fatal(err)
	}
	if _, err := fs.ReadFile(FS, "000001_create_data_sync_tables.down.sql"); err != nil {
		t.Fatal(err)
	}
	up, err := fs.ReadFile(FS, "000001_create_data_sync_tables.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(up)
	if strings.Contains(text, "t_data_sync_schedule") || strings.Contains(strings.ToUpper(text), "FOREIGN KEY") {
		t.Fatalf("stage 1 migration contains an out-of-scope relation: %s", text)
	}
}
