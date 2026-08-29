# API routes

Use an API route when a client needs JSON, no content, or a WebSocket connection rather than an HTML page. API routes live inside the same `web/routes` tree, use the same `*web.Context` and application services, and are registered on the same Go server.

## Map folders to endpoints

```text
web/routes/api/route.go                   /api
web/routes/api/health/route.go            /api/health
web/routes/api/products/route.go          /api/products
web/routes/api/products/id_/route.go      /api/products/{id}
```

A directory may contain one or several Go files. File names are organizational; exported handler names define the methods. A single `route.go` is usually easiest to understand.

## Return JSON

Create `web/routes/api/products/route.go`:

```go
package products

import (
    "net/http"

    "example.test/shop/internal/catalog"
    "github.com/JohnKinyanjui/northframe/pkg/web"
)

func GET(ctx *web.Context) error {
    service := web.MustUse[*catalog.Service](ctx)
    products, err := service.List(ctx.StdContext())
    if err != nil {
        return err
    }
    return ctx.JSON(http.StatusOK, map[string]any{
        "products": products,
    })
}
```

The signature must be exactly `func GET(*web.Context) error`. Supported names are `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS`, and `HEAD`. Northframe discovers them during generation and emits method-qualified `net/http` routes.

## Decode a request body

```go
type createProductInput struct {
    Name       string `json:"name"`
    PriceCents int64  `json:"price_cents"`
}

func POST(ctx *web.Context) error {
    var input createProductInput
    if err := ctx.DecodeJSON(&input); err != nil {
        return web.BadRequest("Invalid product body", err)
    }
    if strings.TrimSpace(input.Name) == "" || input.PriceCents < 0 {
        return web.BadRequest("Name and a non-negative price are required", nil)
    }

    product, err := web.MustUse[*catalog.Service](ctx).
        Create(ctx.StdContext(), catalog.CreateInput{
            Name: input.Name, PriceCents: input.PriceCents,
        })
    if err != nil {
        return err
    }
    return ctx.JSON(http.StatusCreated, product)
}
```

`DecodeJSON` accepts one JSON value, limits the body to 1 MiB, and rejects unknown object fields. This makes accidental client/server contract drift visible. Validation and business rules still belong in application code.

## Dynamic parameters and deletion

Create `web/routes/api/products/id_/route.go`:

```go
package id_

import (
    "github.com/JohnKinyanjui/northframe/pkg/web"
)

func DELETE(ctx *web.Context) error {
    service := web.MustUse[*catalog.Service](ctx)
    if err := service.Delete(ctx.StdContext(), ctx.Param("id")); err != nil {
        return err
    }
    return ctx.NoContent()
}
```

The trailing underscore creates the `{id}` path parameter while keeping a Go-safe directory name.

## Protect one endpoint

Export `Middleware` from the same package:

```go
func Middleware() []web.Middleware {
    return []web.Middleware{
        auth.Require(sessions, auth.GuardOptions{
            Permissions: []string{"products.write"},
        }),
    }
}
```

It must return `[]web.Middleware`. Middleware runs before the API handler and can resolve sessions, enforce permissions, attach locals, apply limits, or add headers.

## API error responses

Return `web.BadRequest`, `web.Unauthorized`, `web.Forbidden`, `web.NotFound`, or `web.Error`. Northframe converts them into a stable JSON error envelope:

```json
{
  "error": {
    "status": 404,
    "message": "Product not found"
  }
}
```

Internal causes are logged for server errors but not serialized. Panics in API handlers are also contained and returned as safe 500 responses.

## WebSocket endpoints

An API directory can export a socket handler instead of `GET`:

```go
func WEBSOCKET(ctx *web.Context, socket *web.Socket) error {
    for {
        var incoming map[string]any
        if err := socket.ReadJSON(socket.Context(), &incoming); err != nil {
            return err
        }
        if err := socket.WriteJSON(socket.Context(), incoming); err != nil {
            return err
        }
    }
}

func WebSocketOptions() web.SocketOptions {
    return web.SocketOptions{
        ReadLimit:      64 << 10,
        ReadTimeout:    30 * time.Second,
        WriteTimeout:   10 * time.Second,
        MaxConnections: 500,
    }
}
```

One directory cannot declare both `GET` and `WEBSOCKET`. Put one of them in a child folder. Same-origin connections are accepted by default; configure explicit origin patterns before allowing cross-origin clients.

## Test the contract

Use `httptest` or a real compiled test server and assert the status, content type, response body, authorization behavior, and rejection of invalid JSON. API routes are still Go HTTP handlers, so standard Go testing tools apply without a browser.
