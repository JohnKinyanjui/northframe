// Package database provides the shared database lifecycle used by Northframe
// applications. Driver packages live below this package so applications only
// compile the database engine they select.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"
)

// Engine is a database dialect understood by Northframe and sqlc.
type Engine string

const (
	SQLite     Engine = "sqlite"
	PostgreSQL Engine = "postgresql"
	MySQL      Engine = "mysql"
)

// Pool controls database/sql connection reuse.
type Pool struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// Configure applies connection-pool settings without opening a connection.
func Configure(db *sql.DB, pool Pool) {
	if pool.MaxOpenConns > 0 {
		db.SetMaxOpenConns(pool.MaxOpenConns)
	}
	if pool.MaxIdleConns > 0 {
		db.SetMaxIdleConns(pool.MaxIdleConns)
	}
	if pool.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(pool.ConnMaxLifetime)
	}
	if pool.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(pool.ConnMaxIdleTime)
	}
}

// Ping verifies that a configured database is reachable.
func Ping(ctx context.Context, db *sql.DB) error {
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	return nil
}

// Migrate applies sorted .sql files exactly once. Separate migration folders
// should be used for different SQL dialects.
func Migrate(ctx context.Context, db *sql.DB, engine Engine, migrations fs.FS) error {
	if err := validateEngine(engine); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `create table if not exists northframe_migrations (
name varchar(255) primary key,
applied_at timestamp not null default current_timestamp
)`); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}

	entries, err := fs.ReadDir(migrations, ".")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		applied, err := migrationApplied(ctx, db, engine, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		source, err := fs.ReadFile(migrations, name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if err := applyMigration(ctx, db, engine, name, string(source)); err != nil {
			return err
		}
	}
	return nil
}

func migrationApplied(ctx context.Context, db *sql.DB, engine Engine, name string) (bool, error) {
	query := "select count(1) from northframe_migrations where name = " + placeholder(engine, 1)
	var count int
	if err := db.QueryRowContext(ctx, query, name).Scan(&count); err != nil {
		return false, fmt.Errorf("inspect migration %s: %w", name, err)
	}
	return count > 0, nil
}

func applyMigration(ctx context.Context, db *sql.DB, engine Engine, name, source string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", name, err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, statement := range splitStatements(source) {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}
	query := "insert into northframe_migrations (name) values (" + placeholder(engine, 1) + ")"
	if _, err := tx.ExecContext(ctx, query, name); err != nil {
		return fmt.Errorf("record migration %s: %w", name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", name, err)
	}
	return nil
}

func splitStatements(source string) []string {
	parts := strings.Split(source, "-- northframe:split")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		statement := strings.TrimSpace(part)
		if statement != "" {
			statements = append(statements, statement)
		}
	}
	return statements
}

func placeholder(engine Engine, index int) string {
	if engine == PostgreSQL {
		return fmt.Sprintf("$%d", index)
	}
	return "?"
}

func validateEngine(engine Engine) error {
	switch engine {
	case SQLite, PostgreSQL, MySQL:
		return nil
	default:
		return errors.New("database engine must be sqlite, postgresql, or mysql")
	}
}
