# Testing

Northframe applications are normal Go programs, so the standard library remains the primary testing tool. Test business services without HTTP, test handlers with `httptest`, and run a smaller number of browser tests against a compiled application for critical user flows.

## Test the service first

Business rules should live outside route sidecars:

```go
func TestCatalogCreateRejectsNegativePrice(t *testing.T) {
    service := catalog.NewService(fakeCatalogStore{})
    _, err := service.Create(context.Background(), catalog.CreateInput{
        Name: "Mug", PriceCents: -1,
    })
    if !errors.Is(err, catalog.ErrInvalidPrice) {
        t.Fatalf("Create error = %v, want ErrInvalidPrice", err)
    }
}
```

This test is fast and does not know about Northframe. Route tests should prove HTTP mapping, validation, status codes, middleware, and rendering—not repeat every domain rule.

## Test the generated application

After `north generate`, register the generated routes on a test app:

```go
func newTestApp(t *testing.T) *web.App {
    t.Helper()

    app := web.New()
    store := catalog.NewMemoryStore()
    service := catalog.NewService(store)
    web.Provide(app, service)
    routes.Register(app)
    return app
}

func TestProductsPage(t *testing.T) {
    app := newTestApp(t)
    request := httptest.NewRequest(http.MethodGet, "/products", nil)
    response := httptest.NewRecorder()

    app.ServeHTTP(response, request)

    if response.Code != http.StatusOK {
        t.Fatalf("status = %d, want 200; body=%s", response.Code, response.Body.String())
    }
    if !strings.Contains(response.Body.String(), "Products") {
        t.Fatalf("page does not contain heading: %s", response.Body.String())
    }
}
```

The test exercises route discovery output, middleware, loaders, layouts, and the generated renderer. Provide the same service types that production code resolves, but use deterministic fakes where a real database is not the subject of the test.

## Test a form action

```go
func TestCreateProductValidation(t *testing.T) {
    app := newTestApp(t)
    body := strings.NewReader("name=&price_cents=-1")
    request := httptest.NewRequest(http.MethodPost, "/products", body)
    request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    request.Header.Set("Accept", "application/json")
    request.Header.Set("X-Northframe-Enhance", "true")
    response := httptest.NewRecorder()

    app.ServeHTTP(response, request)

    if response.Code != http.StatusUnprocessableEntity {
        t.Fatalf("status = %d, want 422; body=%s", response.Code, response.Body.String())
    }
    var result web.ActionResult
    if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
        t.Fatal(err)
    }
    if result.Errors["name"] == "" {
        t.Fatalf("missing name error: %#v", result.Errors)
    }
}
```

If global CSRF middleware is enabled, first make a GET request, retain the CSRF cookie, and send its value as `_northframe_csrf` or `X-CSRF-Token`. Do not disable production security in every test; add focused tests that prove it works.

## Test an API

```go
func TestCreateProductRejectsUnknownJSONField(t *testing.T) {
    app := newTestApp(t)
    request := httptest.NewRequest(
        http.MethodPost,
        "/api/products",
        strings.NewReader(`{"name":"Mug","unexpected":true}`),
    )
    request.Header.Set("Content-Type", "application/json")
    response := httptest.NewRecorder()

    app.ServeHTTP(response, request)

    if response.Code != http.StatusBadRequest {
        t.Fatalf("status = %d, want 400", response.Code)
    }
}
```

Assert the status and public response contract. Also test authentication, wrong methods, malformed bodies, missing records, and safe 500 responses.

## Test sessions

Use `auth.NewMemoryStore()` and inject a deterministic clock through `auth.Config.Clock`. Start a session with an `httptest.ResponseRecorder`, copy its cookie to the request, then assert anonymous, authenticated, expired, and missing-permission cases. The memory store makes session tests isolated without changing production storage.

## Test database queries

Run sqlc integration tests against the same database engine used in production. A SQLite test cannot prove PostgreSQL transaction, locking, JSON, timestamp, or constraint semantics. Apply all migrations to an empty test database, seed only the records needed by the test, and use unique databases or transaction rollback for isolation.

## Compile and race-test

A useful local and CI sequence is:

```sh
north db verify
north generate
go test ./...
go test -race ./...
north deploy check
```

`north generate` catches view contracts before tests compile. `go test ./...` covers both framework-facing and application packages. `north deploy check` proves the production target builds with generated assets.

## Browser tests

Use a real browser only where DOM behavior matters: modal focus, client state, progressive form loading, navigation, and full authentication flows. Start the application on an isolated test port and seed a dedicated database. Prefer stable roles, labels, and visible text over CSS class selectors. Browser tests are valuable, but they should sit on top of service and HTTP tests rather than replacing them.
