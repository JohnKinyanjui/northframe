<p align="center">
  <img src="assets/logo.png" alt="Northframe" width="220">
</p>

<h1 align="center">Northframe</h1>

<p align="center">
  Native Go SSR with typed <code>.north</code> views and one-binary deployment.
</p>

Northframe is an experimental Laravel/Django-style framework for native Go SSR. It compiles structured `.north` views into a small generated Go layer, generates routes from the filesystem, embeds its browser runtime and CSS, and deploys as one executable.

Applications do not need Node, Deno, npm, `package.json`, Svelte, or a JavaScript server runtime.

## Requirements

- Go 1.27 or newer

## Install the command

```sh
go build -o "$(go env GOPATH)/bin/north" ./cmd/cli
north version
```

The CLI provides the application workflow:

```sh
north run
north create .
north generate
north build -o ./app
north db generate
north db create add_store_members
north db migrate
north db rollback
north db seed
north add date-fns
north remove date-fns
north update
north upgrade
north lsp
```

`north run` loads `.env`, compiles the application, and watches `.go`, `.north`, `.css`, `.env`, and public assets. Values already exported by the shell take precedence over `.env`.

`north create .` safely scaffolds the current directory. It preserves unrelated directories, performs a complete conflict preflight before writing, and refuses to overwrite an existing application file.

After installing a newer `north` command, run `north upgrade --check` to preview the generated-file changes, then `north upgrade` to apply them. Upgrade compiles first, touches only `.generated/routes`, validates the Go application, and restores the previous generated files if validation fails. Route files, components, services, database files, and dependency versions are not rewritten.

See [Upgrading a Northframe application](docs/upgrading.md) for the upgrade boundary and its separation from JavaScript dependency updates.

## Application structure

```text
main.go                           # handwritten application entrypoint
app/                              # optional handwritten application code
web/                              # browser and HTTP presentation boundary
├── routes/
│   ├── layout.north             # required root document layout markup
│   ├── layout.north.go          # typed root layout loader and middleware
│   ├── layout.css               # optional colocated CSS
│   ├── page.north               # / markup
│   ├── page.north.go            # typed / loader and actions
│   ├── page.css                 # optional colocated CSS
│   ├── api/                     # optional JSON API routes
│   │   └── products/
│   │       └── route.go         # GET, POST, PUT, PATCH, DELETE...
│   └── auth/
│       ├── layout.north         # optional nested layout
│       ├── layout.north.go      # typed nested layout loader
│       ├── page.north           # /auth
│       ├── page.north.go        # typed /auth loader and actions
│       └── page.css
├── components/                  # independent typed .north components
├── client/                      # shared browser-only TypeScript modules
└── public/                      # assets embedded into the deployment binary
northframe.toml                   # client source and dependency declarations
northframe.lock                   # exact dependency graph and integrity hashes
.generated/routes/               # protected generated tree; never edit or commit
├── root/props_generated.go       # root PageProps/LayoutProps
└── auth/props_generated.go       # mirrors web/routes/auth
```

Every `page.north` requires a sibling `page.north.go`; every `layout.north` requires `layout.north.go`. Route CSS must be named `page.css` or `layout.css`.

`.nf` is intentionally rejected with migration guidance. The extension is now `.north`; it avoids the existing `.nf` ecosystem collision and remains immediately recognizable in a route tree.

## Language server

`north lsp` runs the built-in Language Server Protocol implementation over stdio. A dependency-free VS Code extension is included in `lsp/vscode`; it registers `.north`, starts `north lsp`, and adds syntax highlighting, snippets, restart controls, and server-path configuration.

The server shares the production compiler parser and currently provides:

- live syntax and layout diagnostics
- completion for typed `PageProps`/`LayoutProps` fields and template directives
- hover information and go-to-definition from an expression to its Go props field
- document symbols and formatting

Northframe applications still require no Node process, editor-local parser, or `package.json`. VS Code itself requires a `package.json` manifest inside the extension directory, but that tooling file never enters an application. The colocated `.north.go` file continues to use the normal Go language server.

## What is generated

Northframe does not generate the application. Handlers, services, models, middleware, validation, database code, and business rules remain handwritten Go.

The ignored `.generated/routes` directory contains only:

- native Go render functions compiled from `.north`
- native component prop types, slot fragments, and render functions
- the filesystem route manifest
- layout composition
- generated TypeScript contracts for route and component props
- compiled utility CSS, browser modules, and embedded asset references

Every generated source begins with `// Code generated by Northframe. DO NOT EDIT.` and is recreated automatically by `north run`, `north generate`, and `north build`.

Generated props mirror the handwritten route tree under `.generated/routes`. Handwritten loaders import their protected route contract, while the generated router imports both; this keeps generated code out of application packages without creating an import cycle.

## Generated route props and typed sidecars

For simple rendered values, Northframe infers a string prop directly from the template:

```html
<h1>{Props.Title}</h1>
```

The loader uses the generated type without declaring a struct:

```go
import generated "example.test/app/.generated/routes/customers"

func Page(*web.Context) (generated.PageProps, error) {
    return generated.PageProps{Title: "Customers"}, nil
}
```

Collections, nested models, non-string values, and client-only props need an explicit template contract because their Go type cannot be guessed safely:

```html
<!-- web/routes/users/page.north -->
---
import db "example.test/app/internal/db/generated"

interface Props {
  Users []db.User
}
---

<ul>
  {for user := range Props.Users}
    <li>{user.Name}</li>
  {/for}
</ul>
```

The `.north.go` sidecar remains the source of truth for loading data and can call services, sqlc queries, validation helpers, or any normal Go function:

```go
// web/routes/users/page.north.go
package users

import (
    db "example.test/app/internal/db/generated"
    "northframe.dev/northframe/pkg/web"
)

func Page(ctx *web.Context) (PageProps, error) {
    queries := web.MustUse[*db.Queries](ctx)
    users, err := queries.ListUsers(ctx.StdContext())
    return PageProps{Users: users}, err
}
```

Infrastructure is provided once in `main.go`:

```go
app := web.New()
web.Provide(app, db)
web.Provide(app, queries)
routes.Register(app)
http.ListenAndServe(":8080", app)
```

Northframe emits `PageProps` into the matching `.generated/routes/<route>/props_generated.go`, and the renderer and loader compile against that same type. A renamed, missing, or incompatible field fails at `go build`. `{Props.Name}` is an HTML-escaped Go value. `{if Props.Condition}` and `{for item := range Props.Items}` become native Go control flow and accept Go expressions. Generated renderers are formatted with `gofmt`. Existing handwritten props structs remain supported during migration, but a route must use either the template contract or the handwritten struct—not both.

## Authentication, sessions, and permissions

`pkg/auth` provides optional opaque server-side sessions without imposing a
user table or database engine. Applications implement `auth.Store` with their
database or cache; `auth.NewMemoryStore` is available for development and tests:

```go
sessions, err := auth.New(auth.Config{
    Store:      postgresSessionStore,
    CookieName: "topduka_session",
    Lifetime:   24 * time.Hour,
    Secure:     true,
})

app.Use(auth.Load(sessions))
app.Handle("GET /dashboard", dashboard,
    auth.Require(sessions, auth.GuardOptions{
        LoginPath:   "/login",
        Permissions: []string{"dashboard.read"},
    }),
)
```

After verifying credentials, call `sessions.Start`. The browser receives a
32-byte random HttpOnly opaque token while the store only receives its SHA-256
digest. Sessions support expiry, revocation, application values, exact
permissions, global `*`, and namespace grants such as `orders.*`. Loaders and
actions access the resolved identity with `auth.Current(ctx)` or
`auth.MustCurrent(ctx)`. Password policy, user records, OAuth providers, and
multi-factor challenges remain application services instead of framework-owned
database models.

## Request context and application services

Loaders and actions receive `*web.Context`, not a bare `*http.Request`. The context provides:

- `ctx.Request` for full standard-library access
- `ctx.DB` when a `*sql.DB` was registered with `web.Provide`
- `ctx.Param`, `ctx.Query`, `ctx.Form`, `ctx.FormValue`, and `ctx.Cookie`
- `ctx.StdContext()` for database calls and `ctx.DetachedContext()` for queued or scheduled work
- request-scoped `ctx.Locals`, populated from middleware with `web.SetLocal`
- `web.Use[T](ctx)` and `web.MustUse[T](ctx)` for sqlc query sets, services, mailers, queues, caches, and schedulers
- action-only response helpers including `ctx.Redirect`, `ctx.JSON`, `ctx.NoContent`, and `ctx.SetCookie`

The context does not hard-code one scheduler, queue, or ORM. Applications register the concrete service they chose once, and sidecars resolve it with its real Go type.

## Layouts

The root `web/routes/layout.north` owns the HTML document and must contain `<head>`, `<body>`, and `<slot />`. Northframe generates `LayoutProps`; its sidecar implements `Layout(*web.Context) (LayoutProps, error)`. Subdirectories can add another layout; Northframe wraps the page from its nearest layout outward using native Go fragments.

## Dynamic routes, actions, middleware, and errors

Go-safe parameter folder names compile to standard `net/http` patterns:

```text
web/routes/users/id_/page.north       /users/{id}
web/routes/files/path__/page.north    /files/{path...}
```

Read parameters with `ctx.Param("id")` inside `page.north.go`.

Sidecars can expose a default page action, a relative action, or an absolute application path:

```go
func PageActions() []web.Action {
    return []web.Action{
        web.Post("", saveCurrentPage),
        web.Post("invite", inviteUser),
        web.Post("/signup", signup),
    }
}

func signup(ctx *web.Context) error {
    email := ctx.FormValue("email")
    // validate and call a service or ctx.DB here
    return ctx.Redirect("/account", 303)
}
```

For application forms, prefer a typed action. Northframe decodes the request at
the HTTP boundary and returns field-addressable validation results to enhanced
forms while keeping the same endpoint usable without JavaScript:

```go
type SignupInput struct {
    Email string `form:"email" validate:"required,email"`
    Age   int    `form:"age" validate:"required,min=18"`
}

func PageActions() []web.Action {
    return []web.Action{web.PostForm("/signup", signup)}
}

func signup(ctx *web.Context, input SignupInput) (web.ActionResult, error) {
    if emailAlreadyExists(input.Email) {
        return web.ActionInvalid("Please correct the highlighted field.", web.FieldErrors{
            "email": "An account already uses this email address",
        }), nil
    }
    return web.ActionRedirect("/account", http.StatusSeeOther), nil
}
```

`form` selects the HTML field name, `label` customizes generated messages, and
`validate` supports `required`, `email`, `min`, and `max`. Strings use character
length for min/max; numeric fields use numeric bounds. Custom scalar types can
implement `encoding.TextUnmarshaler`.

Plain HTML routing is the contract: `<form method="post" action="/signup">` works without JavaScript. A button can use the standard `formaction` attribute to select a different path.

`PageMiddleware()` and `LayoutMiddleware()` return `[]web.Middleware`. Layout middleware is inherited by all descendant pages and actions. `web.CSRF()` provides double-submit protection and the embedded browser runtime automatically adds the hidden token to unsafe forms.

Loaders can return `web.NotFound`, `web.BadRequest`, or `web.Error` for controlled status responses. Rendering happens in a buffer, so a loader or renderer failure cannot send half a document.

## API routes

The optional `web/routes/api` directory defines JSON endpoints without templates or
manual router registration. Keep each endpoint in one `route.go` by default:

```text
web/routes/
  api/
    products/
      route.go
    users/
      id_/
        route.go
```

An exported HTTP method function becomes an endpoint at the folder URL:

```go
package products

func GET(ctx *web.Context) error {
    products, err := catalogue.List(ctx.StdContext(), ctx.Query("q"))
    if err != nil {
        return web.Error(http.StatusInternalServerError, "unable to list products", err)
    }
    return ctx.JSON(http.StatusOK, map[string]any{"products": products})
}
```

Northframe discovers `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS`, and
`HEAD`, all with the signature `func METHOD(*web.Context) error`. `web/routes/api`
maps to `/api`; `web/routes/api/products` maps to `/api/products`; `id_` and `path__` retain the
same dynamic and catch-all conventions as page routes. A folder can optionally
export `Middleware() []web.Middleware`.

Northframe discovers method functions across every `.go` file in the endpoint
folder, so large endpoints may still be split later. `get.go` and `post.go` are
an organizational option, never a requirement.

Use `ctx.DecodeJSON(&input)` for strict request decoding. It rejects unknown
fields and limits request bodies to 1 MiB. Controlled errors are returned as a
safe JSON envelope, while internal error details remain server-only.

## WebSockets

WebSockets run on the same Northframe server and through the same middleware
chain as pages and API routes. No proxy or second process is required:

```go
app.WebSocket("/ws/notifications", web.SocketOptions{
    OriginPatterns:   []string{"admin.example.com"}, // omit for same-origin only
    ReadLimit:        64 << 10,
    ReadTimeout:      75 * time.Second,
    WriteTimeout:     5 * time.Second,
    MaxConnections:   1_000,
    Compression:      web.SocketCompressionNoContextTakeover,
}, func(ctx *web.Context, socket *web.Socket) error {
    notifications := web.MustUse[*NotificationService](ctx)
    for {
        var message ClientMessage
        if err := socket.ReadJSON(socket.Context(), &message); err != nil {
            return err
        }
        response, err := notifications.Handle(socket.Context(), message)
        if err != nil {
            return err
        }
        if err := socket.WriteJSON(socket.Context(), response); err != nil {
            return err
        }
    }
}, requireSession)
```

Same-origin verification is enabled by default. `OriginPatterns` explicitly
allows trusted cross-origin browser clients; `InsecureSkipVerify` exists for
non-browser development clients but should not be enabled in production.
Route middleware runs before the upgrade, making it the right place to reject
unauthenticated connections with a normal HTTP response.

`Socket` provides context-aware text, binary, JSON, ping, subprotocol, and close
operations. `ReadLimit`, endpoint connection limits, read/write timeouts, panic
containment, and optional compression are enforced by the framework. During
server shutdown, call `app.ShutdownWebSockets(ctx)` before `http.Server.Shutdown`
to send active clients status 1001 and then force-close them if the deadline
expires.

## Progressive forms and loading UI

Add `nf-enhance` when a POST form should use the embedded browser runtime. Without JavaScript it remains a normal HTML form; with JavaScript, Northframe submits it with `fetch`, follows redirects, preserves CSRF protection, disables submit controls, sets `aria-busy`, and exposes loading/idle content.

```html
<form method="post" action="/signup" nf-enhance>
  <input name="email" type="email" required>
  <button>
    <span nf-idle>Create account</span>
    <span nf-loading hidden>Creating…</span>
  </button>
</form>
```

Forms dispatch `northframe:submit`, `northframe:success`, `northframe:invalid`, `northframe:error`, and `northframe:complete` DOM events for application-specific behavior.

Structured errors are rendered into `[nf-error="field_name"]`; `[nf-message]`
receives the action message. Matching controls receive `aria-invalid`, and the
first invalid field is focused. Template handlers can listen to the hyphenated
aliases such as `on:northframe-invalid={handleInvalid}` and use the generated
`NorthframeActionEvent<T>` TypeScript type.

## Compiled TypeScript state

Route-local browser state lives in a Svelte-like TypeScript block. Northframe checks declarations, assignments, PageProps access, bindings, and event handlers in its supported TypeScript subset, then bundles it with esbuild's Go API and embeds the resulting JavaScript module in the Go application—there is still no Node process, `package.json`, or frontend installation.

```html
<script lang="ts">
let open: boolean = false;
let name: string = "";

function toggle(): void {
  open = !open;
}
</script>

<button type="button" on:click={toggle} aria-expanded={#open}>Menu</button>
<nav show={#open}>Hello {#name}</nav>
<input bind:value={#name}>
```

The namespaces are deliberate: `{Props.Name}` reads typed Go SSR data, while `{#name}` reads TypeScript browser state. The reserved TypeScript `props` value follows the generated `PageProps` or `LayoutProps` contract, is serialized as safe inert JSON during SSR, and is typed in `northframe_contracts_generated.ts`. `on:event={handler}` rerenders after synchronous and asynchronous handlers. `show={#state}`, `bind:value={#state}`, `class:name={#state}`, reactive ARIA/data attributes, and text bindings are compiled rather than interpreted at runtime.

### JavaScript dependencies without package.json

Use normal TypeScript imports after adding a browser package through Northframe:

```sh
north add date-fns
north add chart.js@4.5.0
```

```html
<script lang="ts">
import { format } from "date-fns";
import { calculateTax } from "$client/tax";

let today: string = format(new Date(), "PP");
</script>
```

`northframe.toml` records direct dependencies and the shared client source directory. `northframe.lock` records the exact direct and transitive versions, tarball locations, and integrity hashes. Both files should be committed. Downloaded package contents live under ignored `.northframe/`; `north update` reconstructs them on another machine.

This is intentionally narrower than `package.json`: Go dependencies and the application identity stay in `go.mod`; `northframe.toml` contains only Northframe compiler settings and browser dependencies. It has no Node scripts, lifecycle hooks, package-manager metadata, or Node runtime contract. See [northframe.toml versus package.json](docs/project-manifest.md).

Northframe downloads package metadata and tarballs directly, verifies their published integrity, and never runs package lifecycle scripts. esbuild recursively bundles browser-compatible ESM and CommonJS into the generated Go assets. Relative imports resolve beside the `.north` file, `$client/...` resolves from the configured client directory, and explicit `https://` imports remain browser imports. Node built-ins and native/install-script packages are rejected because the deployed application has no Node runtime. Package-imported CSS is not supported yet; keep styles in `page.css` or `layout.css`.

## Independent components and public assets

Components live below `web/components/`, declare Go props, and are invoked with PascalCase tags:

```html
<!-- web/components/panel.north -->
---
interface Props {
  Title string
}
---

<script lang="ts">
let open: boolean = false;
function toggle(): void { open = !open; }
</script>

<button on:click={toggle}>{Props.Title}</button>
<section show={#open}><slot /></section>
```

Use it from a page, layout, or another component:

```html
<Panel Title={Props.PageTitle}>
  <p>This child markup becomes the component slot.</p>
</Panel>
```

Northframe generates `PanelProps`, `RenderPanel`, and a TypeScript `PanelProps` contract. Go compilation checks prop values, child markup becomes a native `web.Fragment`, and every rendered instance gets an isolated browser-state scope. Missing or unknown props, unknown components, and child content passed to a component without `<slot />` are compiler errors. The earlier `{#include path}` syntax remains available for simple static source inclusion.

Everything below `web/public/` is embedded into the generated router and served from `/public/` with content types, immutable caching, and ETags. No files are required beside the production executable.

## Internal Tailwind-compatible utilities

Common Tailwind utility classes work directly in `.north` views without installing Tailwind:

```html
<main class="max-w-4xl mx-auto px-6 py-16">
  <h1 class="text-5xl font-extrabold text-white">Hello</h1>
</main>
```

Northframe scans views and emits only used utility rules. Colocated `page.css` and `layout.css` remain available for application-specific CSS. The resulting stylesheet is embedded and served at `/_northframe/app.css`.

## Fonts and typography

Northframe exposes four font variables and matching utilities:

```css
/* web/routes/layout.css */
:root {
  --nf-font-sans: "IBM Plex Sans", sans-serif;
  --nf-font-serif: "Source Serif 4", serif;
  --nf-font-mono: "IBM Plex Mono", monospace;
  --nf-font-display: "Fraunces", serif;
}
```

Use them directly in `.north` files:

```html
<body class="font-sans">
  <h1 class="font-display text-5xl">Operations</h1>
</body>
```

For a hosted font, place its stylesheet `<link>` in `web/routes/layout.north`. For a self-hosted font, put the files below `web/public/fonts/` and declare them in `web/routes/layout.css`:

```css
@font-face {
  font-family: "Atlas Sans";
  src: url("/public/fonts/atlas-sans.woff2") format("woff2");
  font-display: swap;
}
```

## Databases and sqlc

Northframe has opt-in driver packages, so an application only compiles the engine it imports:

```go
import (
    "northframe.dev/northframe/pkg/database"
    "northframe.dev/northframe/pkg/database/sqlite"
)

db, err := sqlite.Open("file:app.db?_pragma=foreign_keys(1)", database.Pool{
    MaxOpenConns: 8,
    MaxIdleConns: 4,
})
```

Equivalent packages are available at `database/postgres` and `database/mysql`. The shared package provides pool configuration, health checks, and ordered transactional migrations. Put `-- northframe:split` between statements in a migration.

sqlc supports `postgresql`, `mysql`, and `sqlite`. Each application keeps its own dialect-specific schema and query files, then generates typed Go code with:

```sh
north db generate
```

Applications place their migration runner in `cmd/migrator`. Northframe loads
the project's `.env` and exposes the runner through a consistent CLI:

```sh
north db verify
north db create add_store_members
north db migrate
north db rollback
north db seed
north db status
north db version
```

`north db create` writes the next zero-padded migration below
`internal/db/migrations` with Goose `Up` and `Down` sections. `north db migrate`
and `north db rollback` map to the application runner's `up` and `down`
operations. `north db seed` runs `cmd/seeder` and forwards options such as
`-only`. `north db adopt` is also available for applications that need to
baseline a known legacy schema.

Northframe treats `internal/db/generated` as read-only output. SQL belongs in `internal/db/query`, schemas or migrations belong in `internal/db/schema` or `internal/db/migrations`, and handlers/services consume the generated query API.

## Calculator example

The included calculator demonstrates SSR, a root layout, generated utilities, responsive design, browser hydration, mouse input, keyboard input, chained operations, decimals, percentages, sign changes, backspace, clearing, and division-by-zero handling.

```sh
cd examples/calculator
north run
```

Open <http://localhost:8000>.

Build one deployment executable:

```sh
cd examples/calculator
north build -o ./northframe-calculator
```

## Commerce example

Commerce is the database-backed stress test. It exercises nested routes, independent typed components, isolated component state, Go-to-TypeScript contracts, PostgreSQL migrations and seed data, sqlc generation, SSR, searchable inventory, CSRF-protected actions, pending form UI, and a searchable Northframe comparison Q&A page.

```sh
cd examples/commerce
north db generate
north run
```

Commerce requires `DATABASE_URL`. The Add Product action validates in Go and writes through sqlc; the generated pages, client modules, utility CSS, migrations, and assets ship in the same Go executable. See `examples/commerce/QA.md` for the full regression checklist.

The repository intentionally keeps two focused examples: Commerce for the full framework path and Calculator for rich client state without a database.

## What is still missing

Northframe is now a useful framework prototype, but it is not production-complete. The main remaining systems are:

- credential-provider adapters and an admin resource registry (opaque sessions and permission guards are available now)
- full TypeScript semantic checking beyond Northframe's supported state subset
- JavaScript-package CSS imports, Node built-ins, native addons, lifecycle scripts, and multi-version dependency graphs
- named and multiple component slots, optional/default props, events passed between components, and component package distribution
- serializable action validation results and customizable `.north` error-page conventions
- streaming responses and advanced response metadata
- a complete Tailwind-compatible utility surface, arbitrary values, and diagnostics
- dedicated packaged editor extensions; the stdio LSP server is available now
- production observability, caching, queues, mail, scheduled jobs, and deployment adapters
# northframe
