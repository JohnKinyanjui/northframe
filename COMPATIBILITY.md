# Northframe MVP compatibility contract

Status: **candidate, unversioned**. Northframe must not receive its first release version until every gate below is green and the VS Code extension is accepted by its target registry.

## Supported platform

- Go 1.27 or newer.
- Linux amd64/arm64, macOS arm64, and Windows amd64 for the `north` CLI and generated applications.
- PostgreSQL, MySQL, and SQLite through `pkg/database` and their driver adapters.
- A modern browser with ES modules, CustomEvent, Fetch, and WebSocket support.

## Stable application contract

- Projects use `web/routes`, `web/components`, `web/client`, and `web/public`; protected compiler output lives in root `.generated/routes`.
- `page.north`, `layout.north`, `page.north.go`, and `layout.north.go` define filesystem routes, SSR views, typed loaders, actions, and middleware.
- `routes/api/**/route.go` supports method handlers and `websocket.go` supports WebSocket routes.
- Astro-style `---` frontmatter accepts Go imports and exactly one `interface Props` contract. Exported prop names, Go types, and Go default expressions are type checked.
- SSR uses `{Props.Value}`, `{if ...}`, `{for item := range ...}`, and `{html SafeHTML}`. Browser state uses `<script lang="ts">`, `{#state}`, bindings, and `on:event`.
- Independent PascalCase components support typed props, default props, default/named slots, isolated TypeScript, and bubbling typed custom events.
- Native forms remain usable without JavaScript; `nf-enhance` adds pending, field-error, message, and redirect behavior without changing the Go action contract.

## Stable CLI contract

`north create`, `generate`, `run`, `build`, `db generate/create/migrate/rollback/seed/status/verify`, `add/remove/update`, `upgrade`, `deploy check/docker`, and `lsp` are MVP commands. Command removal or incompatible flag/default changes require an explicit migration note.

## Runtime contract

- Generated SSR escapes ordinary values and accepts raw markup only through `web.SafeHTML`.
- CSRF protection, bounded forms/uploads, safe errors, public-asset caching, WebSocket origin/auth/limit/shutdown behavior, opaque sessions/permissions, and internal admin CRUD remain tested public behavior.
- Optional operations packages expose adapter boundaries for cache, mail, jobs, schedules, logging, and trace correlation. A deployment stays one Go executable and does not require Node at runtime.

## Release gates

Before the first version is assigned:

1. `go test ./...`, `go test -race ./...`, `go vet ./...`, and `govulncheck ./...` pass.
2. Calculator, Commerce, and the production TopDuka acceptance application build.
3. TopDuka migration parity and core interaction tests pass.
4. Linux amd64/arm64, Windows amd64, and macOS arm64 builds pass.
5. `north deploy check` passes on TopDuka.
6. The unversioned VS Code package is installed and tested locally.
7. A publisher credential is supplied and the extension is accepted by Marketplace/Open VSX; only then is the same first version assigned to the CLI/module documentation and extension package.

Until all seven gates pass, `0.0.0` means “unreleased candidate,” not a published semantic version.
