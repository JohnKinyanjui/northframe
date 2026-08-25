package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateProjectPreservesUnrelatedExistingDirectory(t *testing.T) {
	root := t.TempDir()
	reference := filepath.Join(root, "deprecated")
	if err := os.MkdirAll(reference, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reference, "source.txt"), []byte("truth"), 0o644); err != nil {
		t.Fatal(err)
	}
	files := projectScaffold("example.test/shop", "Shop")
	if err := preflightScaffold(root, files); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filepath.Join(reference, "source.txt")); err != nil || string(got) != "truth" {
		t.Fatalf("reference changed: %q, %v", got, err)
	}
}

func TestCreateProjectInExistingRootBuildsProtectedGeneratedTree(t *testing.T) {
	root := t.TempDir()
	reference := filepath.Join(root, "deprecated")
	if err := os.MkdirAll(reference, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reference, "source.txt"), []byte("truth"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := createProject([]string{"--module", "example.test/shop", root}); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"main.go", "web/routes/page.north", "web/routes/api/health/route.go",
		".generated/routes/router_generated.go", ".generated/routes/root/props_generated.go",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(expected))); err != nil {
			t.Errorf("missing %s: %v", expected, err)
		}
	}
	if got, err := os.ReadFile(filepath.Join(reference, "source.txt")); err != nil || string(got) != "truth" {
		t.Fatalf("reference changed: %q, %v", got, err)
	}
}

func TestPreflightScaffoldRejectsConflictingApplicationFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("customer code"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := preflightScaffold(root, projectScaffold("example.test/shop", "Shop"))
	if err == nil || !strings.Contains(err.Error(), "main.go") {
		t.Fatalf("preflight error = %v", err)
	}
}

func TestDefaultModulePath(t *testing.T) {
	if got := defaultModulePath("TopDuka ScaleNodes"); got != "TopDuka-ScaleNodes" {
		t.Fatalf("defaultModulePath = %q", got)
	}
}
