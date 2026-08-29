// Package postgres opens PostgreSQL databases through pgx.
package postgres

import (
	"database/sql"

	"github.com/JohnKinyanjui/northframe/pkg/database"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Open creates a PostgreSQL handle. Use database.Ping to verify connectivity.
func Open(dataSourceName string, pool database.Pool) (*sql.DB, error) {
	db, err := sql.Open("pgx", dataSourceName)
	if err != nil {
		return nil, err
	}
	database.Configure(db, pool)
	return db, nil
}
