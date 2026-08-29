# Error handling

Northframe separates safe HTTP errors from their internal causes. Return helpers such as `web.NotFound`, `web.BadRequest`, `web.Unauthorized`, and `web.Forbidden` from loaders, actions, or handlers. The response receives the intended status and public message while server-side logging retains internal failures.

## Define the application error page

Add `web/routes/error.north`. It is a complete HTML document rather than a normal routed page, so it can still render when a page or layout loader fails:

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>${Props.Status} · Request failed</title>
  </head>
  <body>
    <main>
      <p>Error ${Props.Status}</p>
      <h1>${Props.Message}</h1>
      <p>${Props.Path}</p>
      <a href="/">Return home</a>
    </main>
  </body>
</html>
```

The compiler provides four implicit, escaped props:

- `Status int` is the HTTP response status.
- `Message string` is the safe public message.
- `Path string` is the requested URL path.
- `RequestID string` comes from the `X-Request-ID` request header when present.

Do not declare an `interface Props` in `error.north`; these fields are deliberately fixed so an error page cannot expose an internal cause by accident. The runtime automatically preserves the original status, renders loader and action failures through the page, and uses it for unknown GET routes.

## Return safe errors

```go
func Page(ctx *web.Context) (generated.PageProps, error) {
    product, err := store.Product(ctx.StdContext(), ctx.Param("id"))
    if errors.Is(err, sql.ErrNoRows) {
        return generated.PageProps{}, web.NotFound("Product not found")
    }
    if err != nil {
        return generated.PageProps{}, web.Error(http.StatusInternalServerError, "Unable to load product", err)
    }
    return generated.PageProps{Product: product}, nil
}
```

Never place database errors, credentials, stack traces, or the internal `Cause` in public markup. Nested error boundaries may be added later; in the current contract, `error.north` belongs only at the root of `web/routes`.

Test error pages deliberately: return a known error from a loader in a development-only route, request the page, and verify that the status code and safe message are correct. Keep detailed causes in logs and expose a request ID when your observability middleware provides one. A custom error template is presentation, not authorization; access checks still belong in middleware or the handler.
