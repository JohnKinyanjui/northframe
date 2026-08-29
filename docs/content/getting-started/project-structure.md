# Project structure

Northframe keeps the browser-facing tree together and mirrors it in a protected generated tree. You own `web/`, `main.go`, application packages, migrations, and the two manifest files. The compiler owns `.generated/` and `.northframe/`.

```text
main.go
app/                         # optional handwritten application code
web/
  routes/
    layout.north              # root document layout
    layout.north.go           # layout loader
    page.north                 # route markup
    page.north.go              # route loader and actions
    api/                       # optional API route tree
      health/route.go
  components/                  # reusable .north components
  client/                      # browser-only TS/JS modules
  public/                      # embedded static assets
northframe.toml                # framework and client dependency settings
northframe.lock                # exact browser dependency graph
.generated/routes/             # generated; do not edit
```

The filesystem is the routing convention. `web/routes/page.north` is `/`; `web/routes/auth/page.north` is `/auth`; and `web/routes/products/_id_/page.north` is `/products/{id}`. The sidecar reads that value with `ctx.Param("id")`. API handlers live under `web/routes/api` and use one `route.go` containing whichever HTTP functions the endpoint supports.

## How files become URLs

- `web/routes/page.north` renders `/`.
- `web/routes/sign-up/page.north` renders `/sign-up`.
- `web/routes/products/_id_/page.north` renders `/products/{id}`.
- `web/routes/api/products/route.go` mounts `/api/products`.
- `web/routes/api/products/_id_/route.go` mounts `/api/products/{id}`.

Only reserved route filenames participate in discovery. A page uses `page.north` and `page.north.go`; a layout uses `layout.north` and `layout.north.go`; an API endpoint uses `route.go`. This prevents an unrelated Go helper from becoming public accidentally.

The sidecar is intentionally beside the template. It loads data, declares actions, and can attach middleware while shared business rules remain in services. A request reaches the generated router, the loader receives `*web.Context`, layouts wrap the page, and the renderer writes escaped HTML.

Keep application logic in `.north.go` sidecars, services, and repositories. Generated files are an implementation detail and should never be hand-edited or used as an import location for business logic.

## Where application code belongs

Route sidecars translate HTTP into application calls. Put reusable business rules in domain or service packages, database access behind sqlc queries or repositories, scheduled work in job packages, and shared request middleware in an application package. `main.go` should assemble these dependencies and provide them to `web.App`.

`web/components` is part of the presentation boundary. Organize it by feature when the project grows, for example `web/components/catalog/ProductRow.north`. Its component name becomes `<CatalogProductRow>`, making the source location predictable without a manual import statement.

`web/client` contains TypeScript modules shared by more than one `.north` file. A route-local interaction should stay in that route's `<script lang="ts">`; shared date formatting, charts, or browser API wrappers belong under `web/client` and are imported through `$client/...`.

`web/public` contains files addressed directly by URL. The build embeds them in the Go executable. Keep private source files, credentials, and database dumps outside this directory.

## Common mistakes

Do not put `page.north` in the project root, rename `page.north.go` to an arbitrary filename, or place API handlers beside page markup. Do not commit `.generated` or `.northframe`. If a page needs a shared component, put it under `web/components` so discovery and compiler validation can find it.

## Route CSS

CSS colocated with a route uses the reserved names `page.css` and `layout.css`. Components may have their own styles according to the component build rules. Public files are served from the embedded `web/public` tree.

## Global application CSS

Put application-wide styles in `web/app.css`. Northframe reads this file automatically, appends it after the generated utility layer, and embeds the result in `/_northframe/app.css`. You do not need another `<link>` element, a JavaScript import, or a separate CSS build command.

Use `web/app.css` for shared design tokens, font faces, third-party browser-library styles, and global element defaults. Keep page-only rules in the colocated `page.css` or `layout.css` files so their ownership remains obvious.

```css
:root {
  --brand: #2563eb;
  --surface: #ffffff;
}

body {
  background: var(--surface);
}
```

The generated stylesheet order is: Northframe preflight, compiled utility classes, `web/app.css`, then colocated route CSS.
