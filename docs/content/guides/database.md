# Database and sqlc

Northframe supports PostgreSQL, MySQL, and SQLite through `database/sql`. The framework does not generate an ORM model or hide transactions. Migrations define the schema, sqlc turns handwritten SQL into typed Go, and application services own business rules.

## Organize database code

```text
internal/db/
├── migrations/                  versioned .sql schema changes
├── query/                       SQL read by sqlc
├── generated/                   sqlc output; do not edit
└── migrations.go               embedded migration filesystem
cmd/migrator/                    application migration command
cmd/seeder/                      optional idempotent seed command
sqlc.yaml                        sqlc configuration
```

Commit migrations, query SQL, and `sqlc.yaml`. Never repair a query by editing generated Go; change the SQL and regenerate.

## Open PostgreSQL

```go
db, err := postgres.Open(os.Getenv("DATABASE_URL"), database.Pool{
    MaxOpenConns:    20,
    MaxIdleConns:    10,
    ConnMaxLifetime: 30 * time.Minute,
    ConnMaxIdleTime: 5 * time.Minute,
})
if err != nil {
    log.Fatal(err)
}
defer db.Close()

ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
if err := database.Ping(ctx, db); err != nil {
    log.Fatal(err)
}
```

Use `mysql.Open` or `sqlite.Open` from the matching Northframe package for those engines. Each returns a normal `*sql.DB`. Store connection strings in environment variables or a secret manager, never in templates or generated files.

## Configure sqlc

A PostgreSQL `sqlc.yaml` can look like:

```yaml
version: "2"
sql:
  - engine: "postgresql"
    schema: "internal/db/migrations"
    queries: "internal/db/query"
    gen:
      go:
        package: "dbgen"
        out: "internal/db/generated"
        sql_package: "database/sql"
```

Use `engine: mysql` or `engine: sqlite` for the matching database. SQL syntax and placeholders must match the selected engine.

Create `internal/db/query/products.sql`:

```sql
-- name: ListProducts :many
SELECT id, name, price_cents, active
FROM products
WHERE active = true
ORDER BY name;

-- name: CreateProduct :one
INSERT INTO products (name, price_cents, active)
VALUES ($1, $2, true)
RETURNING id, name, price_cents, active;
```

Generate typed code:

```sh
north db generate
```

Run it whenever migrations or query SQL change. A stale generated package is one of the most common causes of confusing loader compile errors.

## Provide queries or a service

```go
queries := dbgen.New(db)
catalogService := catalog.NewService(queries)

app := web.New()
web.Provide(app, db)
web.Provide(app, queries)
web.Provide(app, catalogService)
```

Use the service from a route:

```go
func Page(ctx *web.Context) (generated.PageProps, error) {
    service := web.MustUse[*catalog.Service](ctx)
    products, err := service.List(ctx.StdContext())
    if err != nil {
        return generated.PageProps{}, web.Error(
            http.StatusInternalServerError,
            "Unable to load products",
            err,
        )
    }
    return generated.PageProps{Products: products}, nil
}
```

Small applications may resolve `*dbgen.Queries` directly. Services become valuable when an operation combines queries, enforces authorization, or owns a transaction.

## Create a migration

```sh
north db create add_products
```

Fill the generated up and down sections:

```sql
-- +goose Up
CREATE TABLE products (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    price_cents BIGINT NOT NULL CHECK (price_cents >= 0),
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE products;
```

This example is PostgreSQL-specific. SQLite and MySQL require engine-appropriate auto-increment, time, boolean, and alter-table syntax.

## Verify and apply migrations

```sh
north db verify
north db status
north db migrate
north db rollback
```

`north db verify` checks migration files without a connection and belongs in CI. The other commands run the application's command at `cmd/migrator` or `cmd/migrate`. That application command owns the embedded migration filesystem, credentials, and target environment.

Northframe also exposes a small embedded migration helper:

```go
//go:embed migrations/*.sql
var migrationFiles embed.FS

migrations, err := fs.Sub(migrationFiles, "migrations")
if err != nil {
    return err
}
return database.Migrate(ctx, db, database.PostgreSQL, migrations)
```

Run migrations as a deliberate release step. Do not let every replica race to change the schema unless the release design explicitly coordinates it.

## Use transactions in services

```go
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()

qtx := queries.WithTx(tx)
if _, err := qtx.CreateOrder(ctx, createParams); err != nil {
    return err
}
if err := qtx.ReserveInventory(ctx, reserveParams); err != nil {
    return err
}
return tx.Commit()
```

The service owns the full transaction. A handler should not commit half an operation and then call another service that can fail.

## Seed and test

Add an idempotent command at `cmd/seeder` and run `north db seed`. Production reference data should be reviewed like a migration and must not create duplicates when rerun.

Test sqlc queries against the same engine used in production. SQLite cannot prove PostgreSQL locking, JSON, timestamp, constraint, or transaction behavior. In CI, apply every migration to an empty database, run `north db generate`, execute integration tests, and compile the application.
