# Northframe beta compatibility contract

Status: **v0.1.0-beta**. This is the first public evaluation release. It is not
`v1` and may contain incompatible changes between beta releases, but changes must
be documented and should include actionable migration guidance.

## Supported platform

- Go 1.27 or newer.
- Linux amd64/arm64, macOS arm64, and Windows amd64 are intended targets for the
  `northframe` CLI and generated applications.
- PostgreSQL, MySQL, and SQLite are supported through `pkg/database` adapters.
- A modern browser with ES modules, CustomEvent, Fetch, and WebSocket support.

## Beta application contract

- Projects use `web/routes`, `web/components`, `web/client`, `web/public`, and
  optional `web/app.css`; protected compiler output lives in `.generated/routes`.
- `page.north` and `page.north.go` define a page; `layout.north` and
  `layout.north.go` define a layout. Filesystem directories determine URLs.
- `web/routes/api/**/route.go` exposes method functions such as `GET`, `POST`,
  `PUT`, `PATCH`, and `DELETE` using `*web.Context`.
- Astro-style `---` frontmatter accepts Go imports and one `interface Props`
  contract. The editor resolves available Go imports on save.
- `${expression}` renders escaped Go data during SSR. `{if ...}` and
  `{for item := range ...}` compile to Go control flow. `{html value}` accepts
  only `web.SafeHTML`.
- `<script lang="ts">` owns optional browser code. `#{expression}` reads client
  state, and `on:event={handler}` attaches browser event handlers.
- PascalCase components support typed props, defaults, slots, isolated
  TypeScript state, and bubbling custom events without manual component imports.
- Native forms work without JavaScript. `nf-enhance` and HTMX-style navigation
  progressively improve loading, validation, history, and partial updates while
  retaining ordinary HTTP behavior.

## Beta CLI contract

The beta includes `northframe create`, `generate`, `run`, `build`, `db`,
`add/remove/update`, `upgrade`, `deploy`, and `lsp`. A command can change before
`v1`, but removal or incompatible default changes require release notes and a
migration path.

## Runtime and security boundary

- Ordinary interpolation is escaped; raw markup requires the concrete
  `web.SafeHTML` type.
- Applications remain responsible for authentication policy, authorization,
  validation, database schema, service boundaries, and secret management.
- Northframe provides optional sessions, CSRF, bounded forms/uploads, safe error
  rendering, asset caching, WebSockets, admin resources, cache, mail, jobs, and
  observability adapters without forcing an application database model.
- A production deployment is a normal Go executable and does not require Node,
  Deno, npm, or a JavaScript server runtime.

## Versioning during beta

- Pin exact tags such as `v0.1.0-beta` in repeatable builds.
- Never edit `.generated`; regenerate after upgrading the CLI.
- Use `northframe upgrade --check`, `northframe generate`, `go test ./...`, and
  `northframe deploy check` before accepting a new beta.
- The VS Code extension and CLI should use the same release family. Restart the
  language server and development server after changing the CLI.

## Gates for a stable release

Before `v1.0.0`, Northframe still needs sustained compatibility testing,
cross-platform builds, security review, race and vulnerability checks, editor
registry publication, production migration evidence, performance baselines, and
clear deprecation policy. Passing the beta test suite is evidence that the
current source is coherent; it is not a claim that all stable-release gates are
complete.
