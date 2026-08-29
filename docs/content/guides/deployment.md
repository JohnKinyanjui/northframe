# Deployment

A Northframe deployment is a compiled Go application. Templates, generated renderers, browser modules, utility CSS, and public assets are linked into the executable. Production does not need Node, Deno, a JavaScript server, or source templates.

## Run the production preflight

```sh
north db verify
go test ./...
north deploy check
```

`north deploy check` regenerates the route tree, runs `go build -trimpath` into a temporary directory, and verifies that a non-empty executable was produced. Use `-target ./cmd/server` when the application entrypoint is not the module root.

This checks compilation, not environment connectivity. Database credentials, external providers, and migrations still need an environment-specific release check.

## Build the executable

```sh
north build -o ./bin/app
```

The command regenerates first, then builds the configured Go main package. Test the exact artifact:

```sh
PORT=8000 DATABASE_URL='postgresql://...' ./bin/app
```

The application should read runtime configuration from environment variables or a secret manager. Do not copy `.env` into the production image, embed credentials in `northframe.toml`, or print secrets during startup.

## Generate a Dockerfile

```sh
north deploy docker -output Dockerfile
```

Use `-target ./cmd/server` for a non-root main package. The generated multi-stage image compiles with the Go version declared in `go.mod`, copies only the executable into Alpine, runs as a non-root user, exposes port 8000, and probes `/api/health`.

Review the generated file before adopting it. If the application requires CA certificates, timezone data, image libraries, or a CGO-backed driver, add those dependencies deliberately. The default PostgreSQL, MySQL, and pure-Go SQLite adapters do not require a Node layer.

## Add a real health endpoint

```go
package health

import (
    "context"
    "net/http"
    "time"

    "github.com/JohnKinyanjui/northframe/pkg/database"
    "github.com/JohnKinyanjui/northframe/pkg/web"
)

func GET(ctx *web.Context) error {
    check, cancel := context.WithTimeout(ctx.StdContext(), time.Second)
    defer cancel()
    if err := database.Ping(check, ctx.DB); err != nil {
        return web.Error(http.StatusServiceUnavailable, "Database unavailable", err)
    }
    return ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
```

Decide whether liveness should depend on the database. Many platforms use a simple process liveness endpoint and a separate readiness endpoint that checks required dependencies.

## Run migrations safely

Apply migrations as a controlled release step:

```sh
north db status
north db migrate
```

Back up important data before risky schema changes. Use expand-and-contract changes when old and new application versions may overlap. In a replicated deployment, run one migrator rather than letting every instance race on startup.

## Graceful shutdown

The production server should listen for SIGINT and SIGTERM, stop taking new requests, close WebSockets, stop schedules, drain jobs within a deadline, and then close database connections. `north run` already handles development children gracefully, but production lifecycle remains application code because only the application knows its resources.

## Verify after release

Check health and readiness, login and logout, one protected page, one form mutation, static assets, a representative database query, background job delivery, and error logging. Confirm that request IDs appear in response headers and logs. Monitor response status, latency, job failures, database pool pressure, and resource consumption.

Rollback means deploying the previous executable and, only when safe, reversing the latest schema change. A down migration that destroys data is not automatically a safe rollback plan.
