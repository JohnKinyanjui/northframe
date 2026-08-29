# Props and rendering

Props are the compile-time contract between a handwritten Go loader and server-rendered markup. They are not a runtime dictionary. Northframe generates a real Go struct, makes the loader return it, and compiles every template expression against it. A missing field or wrong type is a build error.

## Declare an explicit contract

Place imports and `interface Props` in the frontmatter of a page or component:

```north
---
import catalog "example.test/shop/internal/catalog"
import "time"

interface Props {
  Title string
  Products []catalog.Product
  GeneratedAt time.Time
  ShowDrafts bool
}
---
```

Frontmatter imports are Go imports. They use normal Go alias syntax and must resolve through the application's `go.mod`. The VS Code extension can complete packages, hover imported types, navigate to their definitions, and organize imports on save.

You can write a qualified field before its import exists in the file:

```north
---
interface Props {
  ID uuid.UUID
  CreatedAt time.Time
}
---
```

On save, the language server searches the current module, standard library, and dependencies already declared in `go.mod`, adds unambiguous missing imports, removes duplicates, and sorts by import path. If two packages expose the same alias, Northframe prefers the current module and explicitly required modules; it leaves genuinely ambiguous cases unchanged so you can choose the path. Saving never runs `go get` or mutates `go.mod`.

Northframe emits the matching generated type. A route loader returns that type:

```go
func Page(ctx *web.Context) (generated.PageProps, error) {
    service := web.MustUse[*catalog.Service](ctx)
    products, err := service.List(ctx.StdContext(), catalog.ListFilter{
        IncludeDrafts: ctx.Query("drafts") == "true",
    })
    if err != nil {
        return generated.PageProps{}, err
    }
    return generated.PageProps{
        Title:       "Products",
        Products:    products,
        GeneratedAt: time.Now(),
        ShowDrafts:  ctx.Query("drafts") == "true",
    }, nil
}
```

The generated import path mirrors the route under `.generated/routes`. Components receive their own generated props internally; application code normally passes component props in markup rather than constructing those types.

## When inference is enough

For a simple string, Northframe can infer a prop from `${Props.Title}`. Explicit contracts are still recommended when a page has several fields, any non-string value, a collection, an imported application model, or a field used by TypeScript. A visible contract is easier to review and gives the language server complete type information.

Do not declare a second handwritten `PageProps` struct in the sidecar when the template has a contract. The template contract and generated type are one source of truth.

## Render server values

Use `${...}` for server values:

```north
<h1>${Props.Title}</h1>
<time datetime="${Props.GeneratedAt.Format("2006-01-02")}">
  ${Props.GeneratedAt.Format("2 January 2006")}
</time>
<a href="/products/${product.ID}">${product.Name}</a>
```

The expression inside `${...}` is Go. You can access fields, call methods, index values, and use normal operators supported by the compiler. The result is HTML-escaped before it enters the response. User input containing `<script>` is displayed as text rather than executed.

Attribute interpolation and text interpolation use the same escaping rule. Do not manually escape values before passing them as ordinary props or they may be escaped twice.

## Conditions and collections

Server control flow is Go-shaped:

```north
{if len(Props.Products) == 0}
  <EmptyState Title="No products" />
{else}
  <ul>
    {for product := range Props.Products}
      <li>
        <a href="/products/${product.ID}">${product.Name}</a>
        {if product.Active}
          <span>Active</span>
        {else}
          <span>Draft</span>
        {/if}
      </li>
    {/for}
  </ul>
{/if}
```

The loop variable has the element type of the Go slice. Hovering `product` or one of its fields in VS Code should show that type, and go-to-definition can navigate to imported model declarations.

Keep block markers on one logical line. Formatting may indent the HTML body, but it does not split `{for product := range Props.Products}` across lines.

## Pass values to components

Literal strings are quoted. Typed expressions use `${...}`:

```north
<ProductCard
  Label="Featured"
  Product=${product}
  Compact=${true}
/>
```

Northframe checks the attributes against the component's `interface Props`. Unknown attributes and incompatible values fail during compilation. String literals do not need interpolation.

## Render trusted rich HTML

Normal interpolation must remain the default. When an application deliberately renders sanitized CMS content, declare `web.SafeHTML` and use the explicit HTML directive:

```north
---
import "github.com/JohnKinyanjui/northframe/pkg/web"

interface Props {
  ArticleHTML web.SafeHTML
}
---

<article>{html Props.ArticleHTML}</article>
```

Create the value only after sanitizing it with an application allow-list:

```go
clean := sanitizer.Sanitize(untrustedHTML)
props.ArticleHTML = web.SafeHTMLFromSanitized(clean)
```

Northframe does not accept a plain string in `{html ...}`. That type boundary prevents an accidental escape bypass.

## Server props and browser state

Server props are fixed for one HTML response. Browser state belongs in TypeScript:

```north
<script lang="ts">
let expanded: boolean = false;
</script>

<h1>${Props.Title}</h1>
<button type="button" on:click={() => expanded = !expanded}>
  Details
</button>
<section show=#{expanded}>Interactive browser content</section>
```

`${Props.Title}` runs in Go during SSR. `#{expanded}` runs in the browser and can change without a full navigation. A browser variable cannot be referenced with `${...}`, and changing a Go prop does not happen until a request produces another response.

## Debug contract failures

If the compiler says a prop is unknown, check capitalization first: Go fields are case-sensitive. If a type is unresolved, add its package import in frontmatter and ensure the module is present in `go.mod`. If a component prop must use a literal or expression, quote a constant string or wrap a Go value with `${...}`. Never repair a contract by editing `.generated`; correct the template or loader and regenerate.
