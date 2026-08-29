# Routes and layouts

Northframe routes are ordinary directories. The route tree tells the compiler which URL exists, which Go loader supplies its data, which layouts wrap it, and which CSS belongs to it. There is no separate routing configuration to keep synchronized.

## How files become URLs

A directory is a page only when it contains both `page.north` and `page.north.go`:

```text
web/routes/page.north                         /
web/routes/about/page.north                   /about
web/routes/products/page.north                /products
web/routes/products/id_/page.north            /products/{id}
web/routes/shops/shop_id_/orders/id_/page.north
                                              /shops/{shop_id}/orders/{id}
```

Northframe uses `id_` instead of bracket syntax because the directory must also be a valid Go package name. Read a dynamic value with `ctx.Param("id")`:

```go
func Page(ctx *web.Context) (generated.PageProps, error) {
    id := ctx.Param("id")
    product, err := catalog.Product(ctx.StdContext(), id)
    if errors.Is(err, sql.ErrNoRows) {
        return generated.PageProps{}, web.NotFound("Product not found")
    }
    if err != nil {
        return generated.PageProps{}, web.Error(
            http.StatusInternalServerError,
            "Unable to load product",
            err,
        )
    }
    return generated.PageProps{Product: product}, nil
}
```

The route URL is determined by its location, not by the package declaration inside the sidecar. Moving a route directory changes its public URL.

## The `*.north.go` sidecar is required

A `page.north` file is not a complete route by itself. Its sibling `page.north.go` is handwritten application code and must export this loader:

```go
func Page(*web.Context) (generated.PageProps, error)
```

A `layout.north` file must have `layout.north.go` beside it, exporting:

```go
func Layout(*web.Context) (generated.LayoutProps, error)
```

Northframe parses these files during generation. A missing sidecar, wrong function name, missing `*web.Context` parameter, or wrong return values is a compile error. The sidecar is not generated and is never rewritten by `north generate` or `north upgrade`.

The two files have deliberately different responsibilities:

- `*.north` declares the props contract, HTML, server expressions, server control flow, CSS classes, and optional browser TypeScript.
- `*.north.go` reads the request, resolves services, queries data, checks request-specific rules, returns props, and declares middleware or actions.
- `.generated/routes/**` contains the compiler output that connects them. Never move application logic there.

## The page pair

The template is presentation:

```north
---
import catalog "example.test/shop/internal/catalog"

interface Props {
  Product catalog.Product
}
---

<article>
  <h1>${Props.Product.Name}</h1>
  <p>${Props.Product.Description}</p>
</article>
```

The sidecar is the HTTP and application boundary:

```go
package id_

import (
    generated "example.test/shop/.generated/routes/products/id_"
    "example.test/shop/internal/catalog"
    "github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(ctx *web.Context) (generated.PageProps, error) {
    service := web.MustUse[*catalog.Service](ctx)
    product, err := service.Product(ctx.StdContext(), ctx.Param("id"))
    return generated.PageProps{Product: product}, err
}
```

Keep SQL and business rules in services or generated query packages. The loader should translate an HTTP request into a service call and return view data.

## Anatomy of `page.north.go`

A representative page sidecar can contain a loader, route middleware, actions, input types, and private helper functions:

```go
package id_

import (
    "errors"
    "net/http"

    generated "example.test/shop/.generated/routes/products/id_"
    "example.test/shop/internal/catalog"
    "github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(ctx *web.Context) (generated.PageProps, error) {
    service := web.MustUse[*catalog.Service](ctx)
    product, err := service.Product(ctx.StdContext(), ctx.Param("id"))
    if errors.Is(err, catalog.ErrNotFound) {
        return generated.PageProps{}, web.NotFound("Product not found")
    }
    if err != nil {
        return generated.PageProps{}, web.Error(
            http.StatusInternalServerError,
            "Unable to load product",
            err,
        )
    }
    return generated.PageProps{Product: product}, nil
}

func PageActions() []web.Action {
    return []web.Action{
        web.PostForm[UpdateProductInput]("", updateProduct),
    }
}

type UpdateProductInput struct {
    Name string `form:"name" validate:"required,max=120"`
}

func updateProduct(
    ctx *web.Context,
    input UpdateProductInput,
) (web.ActionResult, error) {
    service := web.MustUse[*catalog.Service](ctx)
    if err := service.Update(
        ctx.StdContext(),
        ctx.Param("id"),
        input.Name,
    ); err != nil {
        return web.ActionResult{}, err
    }
    return web.ActionRedirect(
        "/products/"+ctx.Param("id"),
        http.StatusSeeOther,
    ), nil
}
```

Only `Page` is mandatory. `PageMiddleware` and `PageActions` are optional conventions discovered by the compiler. Private helpers and types are ordinary Go and can be split into other `.go` files in the same package when the sidecar becomes long.

The `generated` import points to the protected props package matching the route directory. For `web/routes/products/id_`, the default path is `.generated/routes/products/id_`. The template's `interface Props` becomes `generated.PageProps`, so changing the contract makes incompatible loader code fail during `go build`.

Use the Go package name that belongs to the route directory. The root route normally uses `package routes`. A directory containing hyphens may use the underscore form as its valid Go package name, as the documentation routes do.

## Loader results, errors, and redirects

The page loader does not write the response. It returns props or an error:

```go
func Page(ctx *web.Context) (generated.PageProps, error) {
    if ctx.Query("legacy") == "1" {
        return generated.PageProps{}, web.Redirect(
            "/products",
            http.StatusMovedPermanently,
        )
    }

    products, err := service.List(ctx.StdContext())
    if err != nil {
        return generated.PageProps{}, err
    }
    return generated.PageProps{Products: products}, nil
}
```

Use `web.Redirect(...)` from a page or layout loader. `ctx.Redirect(...)` is for actions and API handlers that own a response writer; calling it from a loader returns an error.

Return `web.NotFound`, `web.BadRequest`, `web.Unauthorized`, `web.Forbidden`, or `web.Error` when the status and safe public message matter. An ordinary internal error becomes a safe 500 response. Northframe buffers the rendered page before committing it, so a loader or renderer failure is handled by the root `error.north` page rather than leaving half an HTML document in the response.

## Root and nested layouts

The root `web/routes/layout.north` is required. It owns the complete HTML document and must include `<head>`, `<body>`, and `<slot />`:

```north
---
interface Props {
  Title string
  SignedIn bool
  AccountName string
}
---

<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>${Props.Title}</title>
  </head>
  <body>
    <header>
      <a href="/">Store</a>
      {if Props.SignedIn}
        <span>${Props.AccountName}</span>
      {/if}
    </header>
    <slot />
  </body>
</html>
```

Its sibling loader runs for every rendered page:

```go
package routes

import (
    "strings"

    generated "example.test/shop/.generated/routes/root"
    "github.com/JohnKinyanjui/northframe/pkg/auth"
    "github.com/JohnKinyanjui/northframe/pkg/web"
)

func Layout(ctx *web.Context) (generated.LayoutProps, error) {
    session, signedIn := auth.Current(ctx)
    title := "Store"
    if ctx.Path() != "/" {
        title = "Store · " + strings.TrimPrefix(ctx.Path(), "/")
    }
    return generated.LayoutProps{
        Title:      title,
        SignedIn:   signedIn,
        AccountName: session.Values["name"],
    }, nil
}
```

`layout.north.go` follows the same contract as a page sidecar, but returns `generated.LayoutProps` from `Layout`. It is the right place to prepare shared presentation data used by the layout: account navigation, store identity, document metadata, feature visibility, or locale. It is not the right place to load every descendant page's table data.

Layouts do not declare `LayoutActions`. Mutations belong to a page's `PageActions` or to an API route. A layout may declare `LayoutMiddleware` because authentication or tenant resolution often applies to an entire subtree.

A nested layout applies only to its directory and descendants:

```text
web/routes/layout.north                    wraps every page
web/routes/page.north                      /
web/routes/dashboard/layout.north          wraps /dashboard/**
web/routes/dashboard/page.north            /dashboard
web/routes/dashboard/orders/page.north     /dashboard/orders
web/routes/login/page.north                /login, root layout only
```

For `/dashboard/orders`, Northframe composes the root layout, dashboard layout, and page in nesting order. Each layout receives its own typed props. Layout loaders should load shared navigation or session presentation data, not unrelated page data.

## Loader and middleware order

For a request to `/dashboard/orders`, execution follows the route tree:

- Root `LayoutMiddleware` runs first.
- Dashboard `LayoutMiddleware` runs next.
- The page's `PageMiddleware` runs last.
- The root `Layout(ctx)` loader prepares root layout props.
- When its generated renderer reaches `<slot />`, the dashboard `Layout(ctx)` loader runs.
- When the nested layout reaches its `<slot />`, `Page(ctx)` loads page props and renders the page.

If an outer middleware rejects the request, no loader runs. If an outer layout loader returns an error, descendant layouts and the page do not run. This makes a layout a real request boundary, not only an HTML wrapper.

Every loader receives the same request-scoped `*web.Context`. Middleware values set with `web.SetLocal` are therefore available to every layout and the page through `ctx.Locals`.

## Route-local middleware

A sidecar can expose middleware for one route subtree:

```go
func LayoutMiddleware() []web.Middleware {
    sessions := applicationSessions()
    return []web.Middleware{
        auth.Require(sessions, auth.GuardOptions{
            LoginPath:   "/login",
            Permissions: []string{"dashboard.read"},
        }),
    }
}
```

Middleware declared by a layout protects the layout subtree. Middleware declared by a page protects that page and its actions. Global concerns such as request logging belong in `app.Use(...)` during startup.

## Route actions

A page may colocate mutations with its loader:

```go
func PageActions() []web.Action {
    return []web.Action{
        web.PostForm[CreateProductInput]("", createProduct),
        web.Delete("id_", deleteProduct),
        web.Post("/inventory/import", importInventory),
    }
}
```

An empty action path uses the current page URL. A relative path is appended to it. A path beginning with `/` is absolute. Actions are standard HTTP endpoints, so forms, `curl`, mobile clients, and enhanced TypeScript can all call them.

## API routes

JSON APIs live below `web/routes/api` and use `route.go`:

```text
web/routes/api/health/route.go              /api/health
web/routes/api/products/route.go            /api/products
web/routes/api/products/id_/route.go        /api/products/{id}
```

API files declare functions such as `GET(ctx *web.Context) error` and `POST(ctx *web.Context) error`. They do not need `page.north` because they return JSON or no content rather than rendered HTML.

## Colocated CSS

Use `page.css` beside `page.north` and `layout.css` beside `layout.north`. Northframe discovers these exact names, includes them in the generated asset graph, and recompiles when they change. Shared public assets belong in `web/public`; shared browser TypeScript belongs in `web/client`.

For styles shared by the whole application, use `web/app.css`. Northframe automatically merges it into the same embedded `/_northframe/app.css` response before colocated route styles. This gives every project one obvious global stylesheet without requiring Node, a CSS import in TypeScript, or another runtime asset server.

## Diagnosing routing problems

When a URL returns 404, check the directory tree first. A missing sidecar means the page pair is incomplete. If Northframe reports an invalid loader signature, compare the function to `func Page(*web.Context) (PageProps, error)` or `func Layout(*web.Context) (LayoutProps, error)`; the props type may be qualified through the generated package. A directory named `[id]` is not a Northframe dynamic route; rename it to `id_`. If the route compiles but a parameter is empty, make sure the name passed to `ctx.Param` matches the directory without its trailing underscore.

Run `north generate` for a one-shot route compile. Compiler diagnostics include the source file and explain missing layouts, unknown components, invalid props, or malformed control flow.
