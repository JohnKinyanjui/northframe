# Forms and actions

Northframe forms start as ordinary HTML. The browser submits a request to a Go action, the action validates typed input and calls application services, and the response either redirects or returns structured errors. Adding `nf-enhance` changes the browser experience without creating a second server contract.

This approach matters for production applications: a form remains usable before JavaScript starts, after a browser extension blocks scripts, and by automated clients that understand normal HTTP.

## The complete request path

When a user submits a form, Northframe follows this sequence:

- The browser uses the form's `method` and `action` attributes.
- The generated router matches a colocated `web.Action`.
- `web.PostForm` parses the request into a Go struct.
- Validation tags produce field-addressable errors before the handler runs.
- The handler checks identity and authorization, then calls a service.
- The action returns success, invalid input, a redirect, or an error.
- A native request follows normal HTTP behavior; an enhanced request updates the page without changing the Go action.

Keep database mutations inside a service or repository. The route action is the HTTP boundary: it should decode input, enforce request-specific security, and translate the service result into an HTTP response.

## Declare typed input

Put the input struct and action registration in the page's `.north.go` sidecar:

```go
package inventory

import (
    "net/http"

    "github.com/JohnKinyanjui/northframe/pkg/web"
)

type AddProductInput struct {
    Name  string `form:"name" label:"Product name" validate:"required,min=2,max=120"`
    SKU   string `form:"sku" label:"SKU" validate:"required,max=40"`
    Stock int    `form:"stock" label:"Opening stock" validate:"min=0"`
}

func PageActions() []web.Action {
    return []web.Action{
        web.PostForm("/inventory/products", addProduct),
    }
}

func addProduct(ctx *web.Context, input AddProductInput) (web.ActionResult, error) {
    catalog := web.MustUse[*CatalogService](ctx)
    if err := catalog.Create(ctx.StdContext(), input.Name, input.SKU, input.Stock); err != nil {
        return web.ActionResult{}, err
    }
    return web.ActionRedirect("/inventory?created=1", http.StatusSeeOther), nil
}
```

`form` selects the incoming field name. Without it, Northframe converts the exported Go field name to snake case. `label` controls the human-readable validation message. Supported validation rules include `required`, `email`, `min`, `max`, and upload-specific limits. Types implementing `encoding.TextUnmarshaler` can define their own textual format.

The action path may be absolute, as above, or relative to the page. An empty path mounts the action on the page URL. Register multiple actions when a screen has distinct operations such as create, archive, and delete.

## Build the native form

The markup uses the same field names declared by the input struct:

```html
<form method="post" action="/inventory/products">
  <label>
    Product name
    <input name="name" required minlength="2" maxlength="120" />
  </label>

  <label>
    SKU
    <input name="sku" required maxlength="40" />
  </label>

  <label>
    Opening stock
    <input name="stock" type="number" min="0" value="0" required />
  </label>

  <button type="submit">Add product</button>
</form>
```

Browser validation improves feedback, but it is not a security boundary. A client can bypass HTML attributes, so the Go input must enforce the same important constraints.

Test this version before enhancing it. Submitting valid input should mutate the database and return a `303 See Other`; refreshing the destination page should not repeat the POST.

## Add pending and error states

Add `nf-enhance` when the native flow works:

```html
<form method="post" action="/inventory/products" nf-enhance>
  <label>
    Product name
    <input name="name" required nf-error="name" />
  </label>

  <label>
    Opening stock
    <input name="stock" type="number" min="0" required nf-error="stock" />
  </label>

  <p nf-error="form" nf-message hidden></p>

  <button type="submit">
    <span nf-idle>Add product</span>
    <span nf-loading hidden>Adding product…</span>
  </button>
</form>
```

`nf-idle` is visible while no request is running. `nf-loading` is shown during submission. `nf-error="field_name"` associates a control or message with one field, while `nf-error="form" nf-message` receives the action's general message. Keep `hidden` on elements that should not appear during SSR.

The enhanced runtime sends the `X-Northframe-Enhance` header. `web.PostForm` then serializes `ActionResult` as JSON. Without that header, it produces normal HTTP behavior. You do not maintain separate handlers for these paths.

## Return validation from business rules

Struct tags handle shape and basic limits. Domain rules still belong in Go:

```go
func addProduct(ctx *web.Context, input AddProductInput) (web.ActionResult, error) {
    catalog := web.MustUse[*CatalogService](ctx)

    if catalog.SKUExists(ctx.StdContext(), input.SKU) {
        return web.ActionInvalid(
            "A product with this SKU already exists.",
            web.FieldErrors{"sku": "Choose a unique SKU."},
        ), nil
    }

    if err := catalog.Create(ctx.StdContext(), input.Name, input.SKU, input.Stock); err != nil {
        return web.ActionResult{}, err
    }
    return web.ActionRedirect("/inventory", http.StatusSeeOther), nil
}
```

Use `web.ActionInvalid` for errors the user can correct. Return an ordinary error for unexpected infrastructure failures; the application error renderer handles it. Do not expose database errors, SQL text, stack traces, or credentials as validation messages.

`web.ActionSuccess(message, data)` is useful when the page can remain in place. `web.ActionRedirect(path, status)` works for both native and enhanced requests. Prefer a redirect after create, update, or delete operations.

## File uploads

Use `enctype="multipart/form-data"` and receive `*multipart.FileHeader` or a slice of file headers. Northframe inspects content instead of trusting the browser-provided MIME type:

```go
type UploadInput struct {
    Image *multipart.FileHeader `form:"image" label:"Product image" validate:"required,maxbytes=5242880" accept:"image/png,image/jpeg,image/webp"`
}
```

Generate an application-owned storage key; never use the submitted filename as a destination path. `web.UploadedFilename` returns a display-safe base name, `web.UploadedContentType` detects the content, and `web.SaveUploadedFile` performs an atomic size-limited local write. Production applications commonly hand the stream to object storage instead.

## Security checklist

- Add CSRF middleware to browser routes that accept unsafe methods.
- Load the current identity from the session inside the action or middleware.
- Check authorization for the specific store, record, or operation.
- Pass `ctx.StdContext()` to database and network calls so cancellation propagates.
- Validate identifiers again after reading them from form data or route parameters.
- Use a transaction when one action changes several records atomically.
- Never trust hidden inputs for ownership, prices, roles, or permissions.

## Debugging forms

If the handler never runs, compare the form's `method` and `action` with `PageActions`. If fields arrive empty, compare every HTML `name` with its `form` tag. If enhanced errors do not appear, verify `nf-enhance`, `nf-error`, and `nf-message`, then inspect the request in the browser network panel.

Run `north generate` after changing action registration or a template. A `404` usually means the action path does not match. A `422` means decoding or validation failed. A `500` should go through `error.north` and should be investigated in server logs using the request ID.

For machine-facing JSON, use `web/routes/api/.../route.go`, call `ctx.DecodeJSON`, and return `ctx.JSON`. A form action and an API handler can still call the same service, preserving one set of business rules.
