package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGeneratedChangesOnlyReportsManagedFiles(t *testing.T) {
	directory := t.TempDir()
	writeUpgradeTestFile(t, filepath.Join(directory, "same_generated.go"), "same")
	writeUpgradeTestFile(t, filepath.Join(directory, "changed_generated.go"), "old")
	writeUpgradeTestFile(t, filepath.Join(directory, "stale_generated.go"), "stale")
	writeUpgradeTestFile(t, filepath.Join(directory, "handwritten.go"), "keep")

	changes, err := generatedChanges(directory, map[string][]byte{
		"same_generated.go":    []byte("same"),
		"changed_generated.go": []byte("new"),
		"new_generated.go":     []byte("new"),
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"create new_generated.go",
		"refresh changed_generated.go",
		"remove stale_generated.go",
	}
	if !reflect.DeepEqual(changes, want) {
		t.Fatalf("changes = %#v, want %#v", changes, want)
	}
}

func writeUpgradeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
