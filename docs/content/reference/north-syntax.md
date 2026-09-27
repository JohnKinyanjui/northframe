# `.north` syntax reference

This page explains the language implemented by the Northframe compiler. A `.north` file combines server-rendered Go data, optional browser-side TypeScript, and HTML-like markup without turning the application into a Node project.

The most important distinction is where an expression runs:

- `${...}` is evaluated by Go on the server while rendering HTML.
- `{if ...}` and `{for ...}` are also compiled to Go and run on the server.
- `#{...}` is evaluated by TypeScript in the browser after the page loads.

Start with server rendering. Add a `<script lang="ts">` block only when an interaction genuinely needs to happen without a navigation, such as opening a dialog or updating a counter.

## Anatomy of a file

A typical component has three regions. Only the markup is required:

```html
---
interface Props {
  Title string
}
---

<script lang="ts">
let expanded: boolean = false;
</script>

<article>
  <h2>${Props.Title}</h2>
  <button type="button" on:click={() => expanded = !expanded}>Details</button>
  <div show=#{expanded}><slot /></div>
</article>
```

Frontmatter is Go-facing and defines the input contract. The script is browser-facing and contains TypeScript state and handlers. The markup connects both worlds while keeping their expression sigils visibly different.

## Props frontmatter

Frontmatter is delimited by `---` and may contain Go imports followed by exactly one `interface Props` declaration:

```html
---
import db "example.test/shop/internal/db/generated"

interface Props {
  Title string
  Products []db.Product
  Count int = 0
}
---
```

Imports use normal Go syntax. Fields use Go types; defaults are Go expressions. The generated route or component contract is emitted under `.generated`. A route sidecar returns the generated props type, while a component receives its props from the parent invocation.

The `Props` name is fixed so every Northframe file has the same obvious input object. Imported package types work like they do in Go, including aliases. For example, `ID uuid.UUID` requires `import uuid "github.com/google/uuid"`. The compiler and language server use the application's `go.mod` to resolve that package.

Defaults are most useful for reusable components:

```html
---
interface Props {
  Label string
  Tone string = "neutral"
  Disabled bool = false
}
---
```

A caller must provide fields without defaults. It may omit fields with defaults.

## Go server output

Use `${expression}` for a Go value rendered during SSR:

```html
<h1>${Props.Title}</h1>
<a href="/products/${product.ID}">${product.Name}</a>
```

Interpolated values are HTML-escaped. Use `{html expression}` only with a `web.SafeHTML` value produced from application-sanitized HTML. Ordinary strings are rejected at this trusted boundary.

Interpolation works in text and attributes. Do not wrap a whole server expression in quotation marks when its value is not a string literal:

```html
<input value="${Props.Query}" />
<MetricCard Count=${Props.Count} />
```

The first form builds an HTML attribute string. The second passes a typed Go value to a component prop.

## Go control flow

Conditions and loops use Go-shaped blocks:

```html
{if Props.Count == 0}
  <p>Empty.</p>
{else}
  {for product := range Props.Products}
    <p>${product.Name}</p>
  {/for}
{/if}
```

The expression is compiled into native Go control flow. Keep the opening directive and its expression in the supported form; Svelte-style `{#if}` and `{#each}` blocks are not the server syntax.

An `if` may use the same comparisons and boolean operators you would write in Go. A `for` introduces its loop variable inside the block. Nesting is allowed as long as every block has the matching closing directive:

```html
{for section := range Props.Sections}
  <section>
    <h2>${section.Title}</h2>
    {if len(section.Items) > 0}
      {for item := range section.Items}
        <p>${item.Label}</p>
      {/for}
    {else}
      <p>No items in this section.</p>
    {/if}
  </section>
{/for}
```

Keep a directive on one logical line. The formatter preserves block indentation and will not split `for item := range Props.Items` into malformed words.

## Components and slots

Files under `web/components` become PascalCase tags based on their filename. Props are typed and checked:

```html
<MetricCard Label="Orders" Value=${Props.OrderCount} />
```

Child content is available through `<slot />`. Named slots use `<slot name="actions" />` and a `<Fragment Name="actions">...</Fragment>` child. Unknown components, props, duplicate slots, and children passed to a component without a slot are compile errors.

Component names come from their path below `web/components`. For example, `web/components/dashboard/MetricCard.north` is invoked as `<DashboardMetricCard />`. This makes component imports automatic: authors do not write a Go or TypeScript import just to render another local `.north` component.

```html
<DashboardMetricCard Label="Revenue" Value=${Props.Revenue}>
  <Fragment Name="actions">
    <a href="/reports/revenue">View report</a>
  </Fragment>
</DashboardMetricCard>
```

## Layouts

`web/routes/layout.north` wraps the route tree. Put `<slot />` where the child page belongs. A nested `layout.north` wraps only its directory and descendants. Layout data comes from the sibling `layout.north.go` loader.

## TypeScript and browser state

Use one optional script block with `lang="ts"` for browser code:

```html
<script lang="ts">
let open: boolean = false;
function toggle(): void { open = !open; }
</script>
<button type="button" on:click={toggle}>Menu</button>
<aside show=#{open}>Navigation</aside>
```

`#{expression}` is TypeScript/client state. It is distinct from `${expression}`, which is Go/SSR. Supported reactive forms include text, `show`, `bind:value`, `class:name`, and ARIA/data attributes. Event handlers use `on:event={handler}`. Client state is bundled for the browser and does not persist to Go unless a form or API request sends it.

Use explicit TypeScript types. They improve editor completion and catch accidental state changes before the browser bundle is emitted:

```html
<script lang="ts">
let query: string = "";
let saving: boolean = false;

function clear(): void {
  query = "";
}
</script>

<input bind:value=#{query} aria-label="Search" />
<button type="button" on:click={clear} disabled=#{saving}>Clear</button>
<p show=#{query.length > 0}>Searching for #{query}</p>
```

Server props may seed generated client data when supported by the component contract, but browser mutations remain local to that page instance. To change durable application state, submit a form or call an API route.

## Forms and actions

Use ordinary forms with an action path:

```html
<form method="post" action="/signup">
  <input name="email" required />
  <button type="submit">Create account</button>
</form>
```

The route sidecar can register `web.Post`, `web.PostForm`, `web.Put`, `web.Patch`, or `web.Delete`. `PostForm` decodes a typed input struct and returns structured validation errors. Enhanced browser submissions use the same action contract; native form navigation remains the fallback.

The action URL decides which Go handler receives the submission; it does not have to be the current page. That keeps forms compatible with normal browser behavior and makes them work before any client script starts. Name every submitted control because Go decoders receive form fields by name.

## Routes and APIs

`web/routes/page.north` maps to `/`; subdirectories map to URL segments. `web/routes/api/<name>/route.go` contains API handlers such as `GET(ctx *web.Context) error` and can return `ctx.JSON`. Page routes use `page.north.go`; API routes use `route.go`.

Dynamic segments use underscored directory names. For example, `web/routes/products/_id_/page.north` maps to `/products/{id}`, and its loader reads the value with `ctx.Param("id")`. Route directories keep the markup, Go loader or action, and optional CSS together.

## Escaping and trusted HTML

Northframe escapes `${...}` values by default, whether they appear in text or attributes. This is the safe path for names, messages, database values, and request-derived content. `{html ...}` deliberately bypasses that escaping and therefore accepts only `web.SafeHTML`. The native documentation renderer returns that type after escaping Markdown input, which is why docs pages can render formatted prose without allowing arbitrary raw HTML.

## Formatting and diagnostics

The Northframe language server understands the frontmatter, TypeScript block, markup tree, server expressions, and component contracts as distinct regions. Format-on-save should preserve readable Go-shaped blocks and TypeScript indentation. Diagnostics report mismatched `{if}` or `{for}` blocks, unknown props or components, invalid expressions, and undeclared client handlers at their `.north` source location.

Multiline tags keep the final bracket beside the last attribute instead of leaving a detached `>` or `/>` line:

```north
<SearchField
  Label="Search activity"
  Placeholder="Search event, aircraft, mission, or location" />

<button
  type="button"
  aria-label="Close activity"
  class="h-9 w-9 rounded-full">
  Close
</button>
```

Formatting is token-safe. Northframe asks the HTML and TypeScript language services for their edits, restores server directives and frontmatter, then rejects the complete result if meaningful Northframe tokens were deleted or duplicated. Go imports inside `---` are resolved and sorted separately during save. The formatter never rewrites the sibling `.north.go`; normal Go tooling owns that file.

When a diagnostic seems stale, save the file and run `northframe generate` from the module root. The command uses the same parser as the language server and reports the authoritative compile error without starting a server.

## Errors

Place a complete HTML document at `web/routes/error.north` to render safe application errors and unknown GET routes. It receives the implicit escaped props `Status int`, `Message string`, `Path string`, and `RequestID string`; do not declare a custom Props interface in this special file. See [Error handling](/reference/errors) for a complete example.
