# Components

Components are reusable `.north` files discovered below `web/components`. They keep markup and optional browser behavior together while routes remain responsible for loading data.

The filename determines the tag’s component name (`metric-card.north` becomes `<MetricCard>`). Components are compiled with the route tree, so they do not need a separate registration call.

## A typed component

```html
---
interface Props {
  Label string
  Value string
}
---

<article class="rounded-xl border p-4">
  <p>${Props.Label}</p>
  <strong>${Props.Value}</strong>
  <slot />
</article>
```

Use the component by its PascalCase name. Attributes passed from the parent are checked against the generated component props type:

```html
<MetricCard Label="Orders" Value=${Props.OrderCount}>
  <a href="/orders">View all</a>
</MetricCard>
```

`<slot />` receives the parent’s default children. Named slots can be declared with `<slot name="actions" />` and supplied with a `<Fragment Name="actions">...</Fragment>` child. A component without a slot must not receive child markup. Unknown components, props, and slots are compile errors.

## Component names and discovery

The path below `web/components` participates in the name. `web/components/MetricCard.north` becomes `<MetricCard>`, while `web/components/catalog/ProductRow.north` becomes `<CatalogProductRow>`. This prevents common filenames from colliding across features and makes Go to Definition predictable.

Local components do not need import statements. The compiler discovers the component tree before compiling routes, validates every invocation, and emits ordinary Go render functions. A missing tag therefore fails during generation rather than appearing as an unknown browser custom element.

Use quoted literals for fixed strings and `${...}` for typed server values:

```html
<CatalogProductRow
  Label="Featured"
  Product=${product}
  Selected=${Props.SelectedProductID == product.ID}
/>
```

Boolean props may also use `{true}` or `{false}`. The prop name is exported and case-sensitive. If VS Code reports that `${expression}` is invalid while `north generate` succeeds, the editor is running an older `north lsp`; reinstall the CLI and restart the language server.

## Defaults and optional input

A prop without a default is required. Declare a Go expression after `=` when callers may omit it:

```html
---
interface Props {
  Label string
  Tone string = "neutral"
  Collapsed bool = false
}
---
```

Defaults are evaluated in the component's Go import scope. This lets a component use values such as `time.Time{}` without forcing every caller to import `time`.

## Named slots

Use named slots when a component owns a stable layout but callers supply distinct regions:

```html
<section>
  <header><slot name="actions" /></header>
  <main><slot /></main>
</section>
```

```html
<PagePanel>
  <Fragment Name="actions"><button>Save</button></Fragment>
  <p>Main panel content</p>
</PagePanel>
```

A slot is a native Go fragment. It does not create an extra client application or hydration boundary.

## Component state and events

A component may have a `<script lang="ts">` block. Each rendered component instance gets an isolated browser-state scope. Use `#{...}` for browser values and `on:event={handler}` for events. Components can dispatch typed browser events to their parent without turning callbacks into Go props.

Keep component imports in the script block and Go imports in the frontmatter contract. Northframe generates native Go render functions and TypeScript contracts; do not edit those generated files.

Start with a component that accepts strings and renders correctly on the first request. Add an explicit Props contract when data becomes structured, then add a script only for behavior that must run in the browser. Run `north generate` after changing a prop so every caller is checked.
