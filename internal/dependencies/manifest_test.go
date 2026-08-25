package dependencies

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindProjectRootPrefersNestedApplication(t *testing.T) {
	moduleRoot := t.TempDir()
	writeManifestTestFile(t, filepath.Join(moduleRoot, "go.mod"), "module example.test/workspace\n")
	appRoot := filepath.Join(moduleRoot, "examples", "commerce")
	if err := os.MkdirAll(filepath.Join(appRoot, "web", "routes", "inventory"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeManifestTestFile(t, filepath.Join(appRoot, "main.go"), "package main\n")

	root, err := FindProjectRoot(filepath.Join(appRoot, "web", "routes", "inventory"))
	if err != nil {
		t.Fatal(err)
	}
	if root != appRoot {
		t.Fatalf("root = %q, want %q", root, appRoot)
	}
}

func writeManifestTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
