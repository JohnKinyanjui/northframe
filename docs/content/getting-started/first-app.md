# Your first application

This walkthrough builds a small product page from an empty Northframe scaffold. By the end, a browser request will travel through a generated route, call handwritten Go, render typed data into HTML, and submit a normal form action. The example is small, but it uses the same boundaries as a production application.

## Create and run the project

Install the CLI and scaffold a project:

```sh
go install github.com/JohnKinyanjui/northframe/cmd/cli@v0.1.0-beta
mkdir hello-north
cd hello-north
north create .
north run
```

Open the address printed by the command, normally `http://localhost:8000`. `north run` compiles the app before starting it, watches source files, and reloads the browser after a successful rebuild. A compile failure is printed in the terminal while the last successful build keeps serving.

The scaffold already contains the root layout, a home route, an API health route, configuration, and an application entrypoint. We will replace the home route.

## Declare the page contract

Edit `web/routes/page.north`:

```north
---
interface Props {
  ProjectName string
  ProductCount int
}
---

<main>
  <p>Northframe</p>
  <h1>${Props.ProjectName} is ready.</h1>

  {if Props.ProductCount == 0}
    <p>No products have been added.</p>
  {else}
    <p>${Props.ProductCount} products are available.</p>
  {/if}

  <form method="post" action="/welcome">
    <label>
      Your name
      <input name="name" required />
    </label>
    <button type="submit">Continue</button>
  </form>
</main>
```

The frontmatter between the `---` markers is a Go-shaped type contract. It tells Northframe which values the template expects. `${Props.ProjectName}` is evaluated on the server and HTML-escaped. The `{if ...}` block is Go code and runs once while the response is rendered.

Northframe generates the corresponding `PageProps` type under `.generated/routes/root`. Do not edit that directory; it is compiler output.

## Load data in Go

Edit the sibling `web/routes/page.north.go`:

```go
package routes

import (
    "net/http"
    "strings"

    generated "hello-north/.generated/routes/root"
    "github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
    return generated.PageProps{
        ProjectName:  "Hello Northframe",
        ProductCount: 3,
    }, nil
}

func PageActions() []web.Action {
    return []web.Action{
        web.Post("/welcome", welcome),
    }
}

func welcome(ctx *web.Context) error {
    name := strings.TrimSpace(ctx.FormValue("name"))
    if name == "" {
        return web.BadRequest("Name is required", nil)
    }
    return ctx.Redirect("/?welcome="+name, http.StatusSeeOther)
}
```

Replace `hello-north` with the module path from `go.mod`. The sidecar is ordinary Go: imports, helper functions, services, tests, and error handling work exactly as they do in any Go package.

The names are conventions:

- `Page` loads server props for a `page.north` file.
- `PageActions` declares non-GET endpoints associated with that page.
- `web.Post` registers a POST action. Absolute paths start from the application root.
- `*web.Context` exposes the request, response, path values, query values, and provided services.

After saving both files, watch the terminal. Northframe regenerates the route, then Go compiles both the generated props and your return value. If `ProductCount` is misspelled or returned as the wrong type, the build fails before a request can reach the page.

If a frontmatter field uses an imported Go type, write the qualified type first and save:

```north
---
interface Props {
  ID uuid.UUID
}
---
```

When `github.com/google/uuid` is already available through `go.mod`, the VS Code extension adds and sorts `import uuid "github.com/google/uuid"` automatically. It does not add new module requirements. Use `go get` deliberately when introducing a dependency, then let save-time import organization update the `.north` contract.

## What happens during a request

When the browser requests `GET /`:

- The generated router matches the filesystem route.
- Northframe creates a `*web.Context` for the request.
- The generated handler calls `Page(ctx)`.
- The page renderer evaluates `{if Props.ProductCount == 0}`.
- Every `${...}` value is escaped and written into the HTML response.
- The root layout wraps the page at its `<slot />` position.

When the form is submitted, the browser sends a normal `POST /welcome`. The action reads the value and returns a redirect. This works without browser JavaScript. Northframe can progressively enhance the same form later, but the HTTP behavior remains the foundation.

## Verify the result

Check all of these before moving on:

- The heading contains `Hello Northframe` rather than the literal interpolation expression.
- The page reports three products.
- Submitting an empty name is blocked by the browser and rejected by the server if sent manually.
- Submitting a name redirects to a URL containing `?welcome=`.
- Changing `ProductCount` to zero renders the empty branch.

You can also verify the route from the terminal:

```sh
curl -i http://localhost:8000/
curl -i -X POST -d 'name=Amina' http://localhost:8000/welcome
```

## Build the deployment binary

Stop the development server and build the real artifact:

```sh
north build -o ./app
PORT=8000 ./app
```

The output is one Go executable containing generated renderers, CSS, the browser runtime, and public assets. Node or Deno is not needed at runtime.

## Common first-app failures

If Northframe reports that no routes exist, run the command from the directory containing `go.mod` and make sure the route tree begins at `web/routes`. If it cannot find a loader, confirm that `page.north.go` is beside `page.north`. If the generated import cannot be resolved, copy the exact module path from `go.mod` and run `north generate` once.

Next, read [Routes and layouts](/concepts/routes-and-layouts) to learn URL mapping and [Props and rendering](/concepts/props) to understand the server contract in depth.
