package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWaitForChangeReturnsInterrupted(t *testing.T) {
	root := t.TempDir()
	interrupts := make(chan os.Signal, 1)
	interrupts <- os.Interrupt

	err := waitForChange(root, filepath.Join(root, ".generated/routes"), "", interrupts)
	if !errors.Is(err, errInterrupted) {
		t.Fatalf("waitForChange() error = %v, want errInterrupted", err)
	}
}

func TestProcessWasInterrupted(t *testing.T) {
	command := exec.Command("sh", "-c", "kill -INT $$")
	err := command.Run()
	if !processWasInterrupted(err) {
		t.Fatalf("processWasInterrupted(%v) = false, want true", err)
	}
}

func TestStopProcessInterruptsChildProcessGroup(t *testing.T) {
	command := exec.Command("sh", "-c", "trap 'exit 0' INT TERM; while :; do sleep 1; done")
	configureChildProcess(command)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	wait := make(chan error, 1)
	go func() { wait <- command.Wait() }()

	stopProcess(command, wait)
	if command.ProcessState == nil {
		t.Fatal("child process was not reaped")
	}
}

func TestDatabaseGenerateRequiresConfig(t *testing.T) {
	err := databaseGenerate([]string{"-config", filepath.Join(t.TempDir(), "missing.yaml")})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("databaseGenerate() error = %v, want missing config error", err)
	}
}

func TestDetectMigratorTargetUsesApplicationConvention(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "cmd", "migrator"), 0o755); err != nil {
		t.Fatal(err)
	}
	target, err := detectMigratorTarget(root)
	if err != nil {
		t.Fatal(err)
	}
	if target != "./cmd/migrator" {
		t.Fatalf("target = %q, want ./cmd/migrator", target)
	}
}

func TestDetectMigratorTargetExplainsMissingConvention(t *testing.T) {
	_, err := detectMigratorTarget(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "cmd/migrator") {
		t.Fatalf("detectMigratorTarget() error = %v, want cmd/migrator guidance", err)
	}
}

func TestResolveRouteImportFromGoMod(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test/app\n\ngo 1.27\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	routes := filepath.Join(root, "web", "routes", "admin")
	if err := os.MkdirAll(routes, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := resolveRouteImport(routes, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "example.test/app/web/routes/admin" {
		t.Fatalf("route import = %q", got)
	}
}

func TestValidateRoutesDirectoryExplainsLegacyLayout(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "routes"), 0o755); err != nil {
		t.Fatal(err)
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	err = validateRoutesDirectory("web/routes")
	if err == nil || !strings.Contains(err.Error(), "move routes, components, client, and public below web/") {
		t.Fatalf("validateRoutesDirectory() error = %v, want migration guidance", err)
	}
}

func TestWatchSignatureIncludesPublicAssets(t *testing.T) {
	root := t.TempDir()
	before, err := watchSignature(root, filepath.Join(root, ".generated/routes"))
	if err != nil {
		t.Fatal(err)
	}
	public := filepath.Join(root, "web", "public")
	if err := os.MkdirAll(public, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(public, "font.woff2"), []byte("font"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := watchSignature(root, filepath.Join(root, ".generated/routes"))
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("public asset did not change watch signature")
	}
}

func TestWatchSignatureIgnoresGeneratedTree(t *testing.T) {
	root := t.TempDir()
	before, err := watchSignature(root, filepath.Join(root, ".generated/routes"))
	if err != nil {
		t.Fatal(err)
	}
	generated := filepath.Join(root, ".generated", "routes", "root")
	if err := os.MkdirAll(generated, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(generated, "props_generated.go"), []byte("package routes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := watchSignature(root, filepath.Join(root, ".generated/routes"))
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatal("generated route props changed the watch signature")
	}
}

func TestAtomicWriteFilePublishesCompleteContents(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "props_generated.go")
	if err := atomicWriteFile(path, []byte("package first\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := atomicWriteFile(path, []byte("package second\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "package second\n" {
		t.Fatalf("contents = %q", contents)
	}
	temporaryFiles, err := filepath.Glob(filepath.Join(directory, ".props_generated.go.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(temporaryFiles) != 0 {
		t.Fatalf("temporary files left behind: %v", temporaryFiles)
	}
}

func TestLoadDotEnvAddsDefaultsWithoutOverridingShell(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	contents := "DATABASE_URL=postgres://file-value\nexport FEATURE_FLAG='enabled'\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := loadDotEnv(path, []string{"DATABASE_URL=postgres://shell-value", "PATH=/bin"})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(result, "\n")
	if strings.Contains(joined, "postgres://file-value") || !strings.Contains(joined, "DATABASE_URL=postgres://shell-value") || !strings.Contains(joined, "FEATURE_FLAG=enabled") {
		t.Fatalf("environment = %q", joined)
	}
}

func TestLoadDotEnvRejectsInvalidAssignment(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("NOT VALID\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadDotEnv(path, nil); err == nil {
		t.Fatal("expected invalid .env assignment to fail")
	}
}

func TestSetEnvironmentReplacesExistingValue(t *testing.T) {
	result := setEnvironment([]string{"PORT=9000", "PATH=/bin"}, "PORT", "8000")
	joined := strings.Join(result, "\n")
	if strings.Contains(joined, "PORT=9000") || !strings.Contains(joined, "PORT=8000") {
		t.Fatalf("environment = %q", joined)
	}
}
