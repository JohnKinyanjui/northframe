package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"northframe.dev/northframe/examples/commerce/.generated/routes"
	db "northframe.dev/northframe/examples/commerce/internal/db/generated"
	"northframe.dev/northframe/examples/commerce/services/commerce"
	"northframe.dev/northframe/pkg/database"
	"northframe.dev/northframe/pkg/database/postgres"
	"northframe.dev/northframe/pkg/web"
)

//go:embed internal/db/migrations/*.sql
var migrationFiles embed.FS

func main() {
	store, closeDatabase := commerceStore()
	defer closeDatabase()

	app := web.New()
	web.Provide[commerce.Store](app, store)
	routes.Register(app)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Relay Commerce running on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, app))
}

func commerceStore() (commerce.Store, func()) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required; add it to .env before running commerce")
	}
	handle, err := postgres.Open(databaseURL, database.Pool{MaxOpenConns: 20, MaxIdleConns: 5, ConnMaxLifetime: time.Hour})
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := database.Ping(ctx, handle); err != nil {
		log.Fatal(err)
	}
	migrations, err := fs.Sub(migrationFiles, "internal/db/migrations")
	if err != nil {
		log.Fatal(err)
	}
	if err := database.Migrate(ctx, handle, database.PostgreSQL, migrations); err != nil {
		log.Fatal(err)
	}
	return db.New(handle), func() { _ = handle.Close() }
}
