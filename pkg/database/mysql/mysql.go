// Package mysql opens MySQL databases for Northframe applications.
package mysql

import (
	"database/sql"

	"northframe.dev/northframe/pkg/database"

	_ "github.com/go-sql-driver/mysql"
)

// Open creates a MySQL handle. Use database.Ping to verify connectivity.
func Open(dataSourceName string, pool database.Pool) (*sql.DB, error) {
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		return nil, err
	}
	database.Configure(db, pool)
	return db, nil
}
