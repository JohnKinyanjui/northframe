# Middleware and security

Middleware wraps an HTTP handler before the route loader, action, or API method executes. Use it for cross-cutting request concerns: authentication, permissions, CSRF protection, request logging, correlation IDs, security headers, tenant resolution, and rate limits.

## Write middleware

A Northframe middleware is the standard Go shape:

```go
func SecurityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        w.Header().Set("X-Frame-Options", "DENY")
        next.ServeHTTP(w, r)
    })
}
```

`web.Middleware` is an alias-compatible function type, so familiar `net/http` patterns and existing middleware can be adapted without a second server.

## Apply middleware globally

```go
app := web.New()
app.Use(
    observability.HTTP(observability.JSONLogger(slog.LevelInfo)),
    SecurityHeaders,
    auth.Load(sessions),
)
```

Global middleware runs for every route registered on the app. Order matters: the first middleware passed to `Use` becomes the outer wrapper and sees the request first and response last.

Use global middleware only for concerns that truly apply to the whole application. A permission guard applied globally could accidentally block login, health, or public asset routes.

## Apply middleware to a route

Page sidecars can export `PageMiddleware` and layouts can export `LayoutMiddleware`. An API package exports `Middleware`:

```go
func PageMiddleware() []web.Middleware {
    return []web.Middleware{
        auth.Require(sessions, auth.GuardOptions{
            Permissions: []string{"products.write"},
            LoginPath:   "/login",
        }),
    }
}
```

A layout guard protects every descendant page and its actions. This is the natural place for a dashboard authentication boundary.

## Resolve request-local values

```go
func Tenant(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        tenant, err := tenants.Resolve(r.Context(), r.Host)
        if err != nil {
            http.Error(w, "Tenant not found", http.StatusNotFound)
            return
        }
        web.SetLocal(r, "tenant_id", tenant.ID)
        next.ServeHTTP(w, r)
    })
}
```

Loaders read `ctx.Locals["tenant_id"]`. Never trust a tenant ID from a form or URL without comparing it to the authenticated session and resolved host.

## Enable CSRF protection

Unsafe cookie-authenticated browser requests require CSRF protection:

```go
app.Use(web.CSRF())
```

The middleware issues a strict SameSite double-submit cookie. Northframe's browser runtime adds the matching hidden `_northframe_csrf` field to non-GET forms. A custom form can render the token explicitly:

```north
<input
  type="hidden"
  name="_northframe_csrf"
  value="${Props.CSRFToken}"
/>
```

Load it in the sidecar with `web.CSRFToken(ctx)`. Enhanced requests may send the same value through the `X-CSRF-Token` header.

CSRF is relevant when the browser automatically sends authentication cookies. A public bearer-token API needs a different threat model, including strict CORS and token handling.

## Escape output by default

`${...}` is HTML-escaped. Use `{html value}` only with `web.SafeHTML` produced after application sanitization. Escaping does not replace validation, authorization, SQL parameterization, safe redirects, or a Content Security Policy, but it removes the default template XSS path.

## Handle errors safely

Return a public message and keep the internal cause:

```go
return web.Error(
    http.StatusInternalServerError,
    "Unable to update the product",
    err,
)
```

The client sees the safe message. Logs retain the wrapped cause. Never return raw database errors, provider responses, credentials, or stack traces to the browser.

## Add request correlation

```go
logger := observability.JSONLogger(slog.LevelInfo)
app.Use(observability.HTTP(logger))
```

The middleware accepts or generates `X-Request-ID`, creates a W3C-compatible trace ID, adds response headers, and logs method, path, status, bytes, duration, request ID, and trace ID. Retrieve the identifiers from `ctx.StdContext()` with `observability.RequestID` and `observability.TraceID`.

## Security review checklist

Protect mutations with both authentication and authorization. Add CSRF for cookie-authenticated forms. Validate and limit request bodies and file uploads. Parameterize SQL through sqlc. Use HTTPS and secure cookies. Keep secrets out of templates, generated files, logs, and the repository. Restrict WebSocket origins. Add rate limiting at the application or edge for login, signup, password reset, uploads, and expensive API methods. Test failure cases with real HTTP requests.
