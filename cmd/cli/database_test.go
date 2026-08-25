package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDatabaseCreateUsesNextContiguousVersion(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "00001_users.sql"), []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "00004_orders.sql"), []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := databaseCreate([]string{"-dir", directory, "Add Store Members"}); err != nil {
		t.Fatalf("create migration: %v", err)
	}
	path := filepath.Join(directory, "00005_add_store_members.sql")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	text := string(contents)
	if !strings.Contains(text, "-- +goose Up") || !strings.Contains(text, "-- +goose Down") {
		t.Fatalf("migration does not contain Goose directions:\n%s", text)
	}
}

func TestMigrationNameIsFilesystemSafe(t *testing.T) {
	tests := map[string]string{
		"Add Store Members":    "add_store_members",
		"  invoices---status ": "invoices_status",
		"Customers.v2":         "customers_v2",
		"!!!":                  "",
	}
	for input, want := range tests {
		if got := migrationName(input); got != want {
			t.Errorf("migrationName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestDetectCommandTargetFindsSeeder(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "cmd", "seeder"), 0o755); err != nil {
		t.Fatal(err)
	}
	target, err := detectCommandTarget(root, []string{"cmd/seeder"}, "missing")
	if err != nil {
		t.Fatal(err)
	}
	if target != "./cmd/seeder" {
		t.Fatalf("target = %q, want ./cmd/seeder", target)
	}
}
