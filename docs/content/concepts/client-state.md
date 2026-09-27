# Client state

Northframe renders the first page in Go and adds TypeScript only where the browser needs interaction. There is no JavaScript server and no WebAssembly requirement. A `<script lang="ts">` block is checked, compiled into a browser module, and embedded in the Go binary.

## Understand the two environments

```north
---
interface Props {
  AccountName string
}
---

<script lang="ts">
let open: boolean = false;
let count: number = 0;

function toggle(): void {
  open = !open;
}

function increment(): void {
  count += 1;
}
</script>

<h1>${Props.AccountName}</h1>
<button type="button" on:click={toggle}>Toggle menu</button>
<aside show=#{open}>Account navigation</aside>
<button type="button" on:click={increment}>Clicked #{count} times</button>
```

`${Props.AccountName}` is evaluated by generated Go during SSR. The browser receives its escaped text. `#{open}` and `#{count}` are reactive TypeScript expressions that update after events.

Use server props for database results, permissions, request data, and initial HTML. Use client state for a dialog being open, a selected tab, temporary input, a local filter, or a loading indicator.

## Handle events

```north
<script lang="ts">
let loading: boolean = false;

function close(): void {
  open = false;
}

async function refresh(): Promise<void> {
  loading = true;
  try {
    await fetch("/api/products");
  } finally {
    loading = false;
  }
}
</script>

<button type="button" on:click={close}>Close</button>
<button type="button" on:click={refresh} disabled=#{loading}>
  <span show=#{!loading}>Refresh</span>
  <span show=#{loading}>Refreshing…</span>
</button>
```

Handlers must be declared in the same script or imported from a shared browser module. A Go function cannot be a click handler because Go is not running in the browser. Call a form action or API route when an event must reach the server.

## Bind inputs and attributes

```north
<script lang="ts">
let query: string = "";
let filtersOpen: boolean = false;
</script>

<input type="search" bind:value={query} />
<p show=#{query.length > 0}>Filtering for “#{query}”</p>

<button
  type="button"
  class:font-bold={filtersOpen}
  aria-expanded=#{filtersOpen}
  on:click={() => filtersOpen = !filtersOpen}
>
  Filters
</button>
<section show=#{filtersOpen} aria-hidden=#{!filtersOpen}>
  Filter controls
</section>
```

`bind:value` keeps an input and variable synchronized. `show` controls visibility while retaining the element. `class:name` toggles a class, and reactive ARIA/data attributes keep accessibility state aligned with the interface.

## Enhance a normal form

Start with a form that works without JavaScript:

```north
<script lang="ts">
let saving: boolean = false;
let message: string = "";

async function submit(event: SubmitEvent): Promise<void> {
  event.preventDefault();
  const form = event.currentTarget as HTMLFormElement;
  saving = true;
  try {
    const response = await fetch(form.action, {
      method: form.method,
      body: new FormData(form),
      headers: {
        "X-Northframe-Enhance": "true",
        "Accept": "application/json",
      },
    });
    const result = await response.json();
    message = result.message ?? "";
    if (result.redirect) window.location.assign(result.redirect);
  } finally {
    saving = false;
  }
}
</script>

<form method="post" action="/products" on:submit={submit}>
  <input name="name" required />
  <button type="submit" disabled=#{saving}>
    <span show=#{!saving}>Save product</span>
    <span show=#{saving}>Saving…</span>
  </button>
  <p aria-live="polite">#{message}</p>
</form>
```

Without JavaScript, the browser performs a full POST and follows the server result. With the enhancement, a `web.PostForm` action returns the stable `web.ActionResult` JSON contract. This preserves one server validation path.

## Import shared browser code

Put reusable TypeScript in `web/client`:

```ts
// web/client/money.ts
export function formatMoney(amount: number, currency: string): string {
  return new Intl.NumberFormat("en", {
    style: "currency",
    currency,
  }).format(amount);
}
```

Import it with the `$client/` alias:

```north
<script lang="ts">
import { formatMoney } from "$client/money";
let preview: string = formatMoney(1250, "KES");
</script>

<output>#{preview}</output>
```

Third-party browser libraries use `northframe add` and bare imports. They are bundled at build time; Node is not part of production.

## Choose state ownership

Keep state in the smallest component that owns the behavior. A dialog owns its open state. A search field owns its temporary query. Persistent account preferences belong on the server and should be updated through an action. Use a shared module or browser event only when distant components genuinely need to coordinate.

If a client expression is displayed literally, check for a TypeScript compile error and confirm it uses `#{...}`. If the language server reports an undeclared handler, verify the name and restart the Northframe language server after updating the CLI or extension.
