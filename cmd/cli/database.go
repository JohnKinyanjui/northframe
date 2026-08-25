package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var migrationFilePattern = regexp.MustCompile(`^(\d+)[_-].*\.sql$`)

func database(arguments []string) error {
	if len(arguments) == 0 {
		return errors.New("db requires a subcommand: generate, create, migrate, rollback, seed, status, version, verify, or adopt")
	}
	switch arguments[0] {
	case "generate":
		return databaseGenerate(arguments[1:])
	case "create":
		return databaseCreate(arguments[1:])
	case "migrate":
		return databaseMigration("up", arguments[1:])
	case "rollback":
		return databaseMigration("down", arguments[1:])
	case "seed":
		return databaseSeed(arguments[1:])
	case "status", "version", "verify", "adopt":
		return databaseMigration(arguments[0], arguments[1:])
	default:
		return fmt.Errorf("unknown db subcommand %q; expected generate, create, migrate, rollback, seed, status, version, verify, or adopt", arguments[0])
	}
}

func databaseCreate(arguments []string) error {
	flags := flag.NewFlagSet("db create", flag.ContinueOnError)
	directory := flags.String("dir", filepath.FromSlash("internal/db/migrations"), "migration directory")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return errors.New("usage: north db create [-dir path] <migration name>")
	}
	name := migrationName(flags.Arg(0))
	if name == "" {
		return errors.New("migration name must contain a letter or number")
	}
	if err := os.MkdirAll(*directory, 0o755); err != nil {
		return fmt.Errorf("create migration directory: %w", err)
	}
	version, err := nextMigrationVersion(*directory)
	if err != nil {
		return err
	}
	path := filepath.Join(*directory, fmt.Sprintf("%05d_%s.sql", version, name))
	contents := "-- +goose Up\n-- Write the forward migration here.\n\n-- +goose Down\n-- Write the rollback migration here.\n"
	if err := atomicWriteFile(path, []byte(contents), 0o644); err != nil {
		return fmt.Errorf("write migration: %w", err)
	}
	fmt.Println("created migration", path)
	return nil
}

func migrationName(raw string) string {
	var result strings.Builder
	underscore := false
	for _, current := range strings.ToLower(strings.TrimSpace(raw)) {
		if current >= 'a' && current <= 'z' || current >= '0' && current <= '9' {
			result.WriteRune(current)
			underscore = false
			continue
		}
		if result.Len() > 0 && !underscore {
			result.WriteByte('_')
			underscore = true
		}
	}
	return strings.Trim(result.String(), "_")
}

func nextMigrationVersion(directory string) (int, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return 0, fmt.Errorf("read migration directory: %w", err)
	}
	latest := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := migrationFilePattern.FindStringSubmatch(entry.Name())
		if len(matches) != 2 {
			continue
		}
		version, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, fmt.Errorf("parse migration version in %s: %w", entry.Name(), err)
		}
		if version > latest {
			latest = version
		}
	}
	return latest + 1, nil
}

func databaseGenerate(arguments []string) error {
	flags := flag.NewFlagSet("db generate", flag.ContinueOnError)
	config := flags.String("config", "sqlc.yaml", "sqlc configuration file")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if _, err := os.Stat(*config); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("sqlc configuration %q not found", *config)
		}
		return fmt.Errorf("inspect sqlc configuration: %w", err)
	}
	sqlc, err := exec.LookPath("sqlc")
	if err != nil {
		return errors.New("sqlc is not installed; run `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`")
	}
	command := exec.Command(sqlc, "generate", "-f", *config)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("sqlc generate: %w", err)
	}
	fmt.Println("generated typed database code from", *config)
	return nil
}

func databaseMigration(operation string, arguments []string) error {
	target, err := detectMigratorTarget(".")
	if err != nil {
		return err
	}
	environment, err := loadDotEnv(".env", os.Environ())
	if err != nil {
		return err
	}
	commandArguments := []string{"run", target, operation}
	commandArguments = append(commandArguments, arguments...)
	command := exec.Command("go", commandArguments...)
	command.Env = environment
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("db %s: %w", operation, err)
	}
	if operation == "up" {
		fmt.Println("database migrations complete")
	}
	return nil
}

func databaseSeed(arguments []string) error {
	target, err := detectCommandTarget(".", []string{"cmd/seeder", "cmd/seed"}, "database seeder not found; add a Go command at cmd/seeder")
	if err != nil {
		return err
	}
	environment, err := loadDotEnv(".env", os.Environ())
	if err != nil {
		return err
	}
	commandArguments := append([]string{"run", target}, arguments...)
	command := exec.Command("go", commandArguments...)
	command.Env = environment
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("db seed: %w", err)
	}
	fmt.Println("database seeds complete")
	return nil
}

func detectMigratorTarget(root string) (string, error) {
	return detectCommandTarget(root, []string{"cmd/migrator", "cmd/migrate"}, "database migrator not found; add a Go command at cmd/migrator")
}

func detectCommandTarget(root string, candidates []string, missingMessage string) (string, error) {
	for _, candidate := range candidates {
		path := filepath.Join(root, candidate)
		info, err := os.Stat(path)
		if err == nil && info.IsDir() {
			return "./" + filepath.ToSlash(candidate), nil
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("inspect database migrator %q: %w", path, err)
		}
	}
	return "", errors.New(missingMessage)
}
