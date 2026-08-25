# Commerce quality checklist

This checklist covers the behaviour that makes the Commerce example a useful Northframe stress test. Automated checks live beside the relevant Go package; browser checks are intentionally phrased as questions with an expected answer.

## Startup and configuration

- Does `north run` load `DATABASE_URL` from `.env`? Yes; shell variables take precedence over file values.
- Does startup fail clearly when `DATABASE_URL` is absent? Yes; Commerce has no silent demo-data fallback.
- Does a bad PostgreSQL address produce an actionable connection error? It should identify database startup rather than failing on the first page request.
- Are migrations applied before routes begin serving? Yes.
- Are migrations idempotent across restarts? Yes; each embedded migration is recorded and runs once.
- Does `north build` produce one executable containing routes, CSS, client modules, migrations, and assets? Yes.
- Can Commerce and Calculator generate independently? Yes; neither example imports the other.

## Server-rendered pages and routes

- Are `PageProps` and `LayoutProps` generated from template contracts rather than repeated in every loader?
- Are generated props protected under `.generated/routes/<matching-route>/props_generated.go` with no generated files beside handwritten routes?
- Does `GET /` render operational order data from PostgreSQL?
- Does `GET /inventory` render product data before JavaScript runs?
- Does `GET /help` render the complete Q&A bank on the server?
- Does `GET /help?q=postgresql` return only matching answers?
- Does an unknown route return 404 instead of the home page?
- Are query values HTML-escaped when echoed into an input?
- Does every page inherit the Relay layout and CSRF middleware?
- Do Orders, Inventory, and Q&A links work with JavaScript disabled?

## Inventory search and status

- Can search match a product by name, SKU, or category?
- Is matching case-insensitive and whitespace-tolerant?
- Does an empty search restore every product?
- Are zero-stock products labelled `out`?
- Are positive products at or below the reorder point labelled `low`?
- Are products above the reorder point labelled `healthy`?
- Are risky products ordered before healthy products?
- Are money values rendered from integer cents without floating-point drift?

## Add Product dialog and client state

- Is the dialog absent from the visual page on the first SSR paint? Yes; it has both the `hidden` attribute and Tailwind's `hidden` class.
- Does Add product remove both hiding mechanisms after hydration?
- Does the backdrop close the dialog?
- Do the close and Cancel buttons close the dialog?
- Does closing clear the client-side product-name preview?
- Does `aria-expanded` follow the dialog state?
- Does the dialog identify itself with `role="dialog"` and `aria-modal="true"`?
- Does the form remain a normal server POST if JavaScript is disabled?
- Does the enhanced form show `Saving…`, disable submit controls, and expose `aria-busy` while pending?

## Validation, security, and persistence

- Are SKU, name, and category required in both browser and Go validation?
- Is the SKU trimmed and normalized to uppercase?
- Are negative stock, reorder point, and price rejected before sqlc is called?
- Are malformed integer and price fields returned as HTTP 400?
- Does PostgreSQL enforce SKU uniqueness even when requests race?
- Does a valid form use sqlc's generated `CreateProduct` query?
- Does a successful insert redirect to `/inventory` with HTTP 303?
- Does the new product appear after the redirect?
- Does a POST without the matching CSRF token return HTTP 403?
- Does the CSRF cookie use `SameSite=Strict` and become Secure under TLS?
- Are database errors wrapped for logs without exposing raw SQL errors to the user?

## Regression commands

From the repository root:

```sh
go test ./...
go test -race ./...
go vet ./...
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/commerce_db go test -count=1 -v ./examples/commerce
```

From `examples/commerce`:

```sh
north db generate
north generate
north build -o ./relay-commerce
```
