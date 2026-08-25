// Package sqlite opens pure-Go SQLite databases for Northframe applications.
package sqlite

import (
	"database/sql"

	"northframe.dev/northframe/pkg/database"

	_ "modernc.org/sqlite"
)

// Open creates a SQLite handle. Use database.Ping to verify connectivity.
func Open(dataSourceName string, pool database.Pool) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, err
	}
	database.Configure(db, pool)
	return db, nil
}
