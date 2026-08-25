package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func database(arguments []string) error {
	if len(arguments) == 0 {
		return errors.New("db requires a subcommand: generate, migrate, status, version, verify, or adopt")
	}
	switch arguments[0] {
	case "generate":
		return databaseGenerate(arguments[1:])
	case "migrate":
		return databaseMigration("up", arguments[1:])
	case "status", "version", "verify", "adopt":
		return databaseMigration(arguments[0], arguments[1:])
	default:
		return fmt.Errorf("unknown db subcommand %q; expected generate, migrate, status, version, verify, or adopt", arguments[0])
	}
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

func detectMigratorTarget(root string) (string, error) {
	for _, candidate := range []string{"cmd/migrator", "cmd/migrate"} {
		path := filepath.Join(root, candidate)
		info, err := os.Stat(path)
		if err == nil && info.IsDir() {
			return "./" + filepath.ToSlash(candidate), nil
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("inspect database migrator %q: %w", path, err)
		}
	}
	return "", errors.New("database migrator not found; add a Go command at cmd/migrator")
}
