# Request context

Every page loader, layout loader, action, and API handler receives `*web.Context`. It keeps the full standard library request available while providing one consistent place for path values, query values, response helpers, application services, and request-local data.

## Read request data

```go
func Page(ctx *web.Context) (generated.PageProps, error) {
    return generated.PageProps{
        Path:   ctx.Path(),
        ID:     ctx.Param("id"),
        Query:  ctx.Query("q"),
        Locale: ctx.Request.Header.Get("Accept-Language"),
    }, nil
}
```

`ctx.Request` is the original `*http.Request`. `ctx.Param("id")` reads a dynamic route value, and `ctx.Query("q")` reads the first matching query parameter. Lower-level form actions can use `ctx.Form()`, `ctx.FormValue(name)`, and `ctx.Cookie(name)`, although typed `web.PostForm` should be preferred for structured input.

## Respect cancellation

Pass `ctx.StdContext()` into sqlc, database, HTTP, queue, and service calls:

```go
orders, err := queries.ListOrders(ctx.StdContext(), storeID)
```

This is the request's standard `context.Context`. Work is cancelled when the client disconnects, the server shuts down, or a deadline expires. Do not replace it with `context.Background()` inside request code.

`ctx.DetachedContext()` retains request values but removes request cancellation. Use it only to enqueue work that must outlive the response. Copy the small values a job needs; never pass `*web.Context` to a goroutine.

## Provide application services

Create long-lived infrastructure once:

```go
queries := dbgen.New(db)
catalogService := catalog.NewService(queries)

app := web.New()
web.Provide(app, db)
web.Provide(app, queries)
web.Provide(app, catalogService)
```

Resolve the concrete type in a loader:

```go
func Page(ctx *web.Context) (generated.PageProps, error) {
    service := web.MustUse[*catalog.Service](ctx)
    products, err := service.List(ctx.StdContext())
    return generated.PageProps{Products: products}, err
}
```

`web.Use[T](ctx)` returns a value and boolean when absence is expected. `web.MustUse[T](ctx)` fails loudly when a required dependency is missing. Dependencies use their exact Go type: providing `*catalog.Service` does not satisfy a request for `catalog.Service`.

A provided `*sql.DB` is also exposed as `ctx.DB`. Prefer a typed sqlc query set or service for application work because it is easier to test and keeps SQL out of HTTP code.

## Attach request-local data

Middleware can resolve a tenant once:

```go
func Tenant(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tenantID := tenantFromHost(r.Host)
        if tenantID == "" {
            http.Error(w, "Tenant not found", http.StatusNotFound)
            return
        }
        web.SetLocal(r, "tenant_id", tenantID)
        next.ServeHTTP(w, r)
    })
}
```

Read it later with a checked assertion:

```go
tenantID, ok := ctx.Locals["tenant_id"].(string)
if !ok {
    return generated.PageProps{}, web.BadRequest("Tenant is required", nil)
}
```

Locals last for one request. They are appropriate for session, tenant, locale, and correlation values—not global mutable state.

## Write responses

Actions and API handlers have a writable context:

```go
func POST(ctx *web.Context) error {
    var input CreateProductInput
    if err := ctx.DecodeJSON(&input); err != nil {
        return web.BadRequest("Invalid product body", err)
    }
    product, err := web.MustUse[*catalog.Service](ctx).
        Create(ctx.StdContext(), input)
    if err != nil {
        return err
    }
    return ctx.JSON(http.StatusCreated, product)
}
```

Response helpers include `ctx.JSON`, `ctx.Redirect`, `ctx.NoContent`, and `ctx.SetCookie`. Loaders do not own a response writer. Redirect from a loader by returning `web.Redirect(path, status)` as its error.

## Test context-dependent code

Register dependencies and middleware on a `web.App`, register generated routes, and make an `httptest` request. This uses the same private request state as production and is safer than constructing `web.Context` manually. Test services independently wherever HTTP details are not relevant.
