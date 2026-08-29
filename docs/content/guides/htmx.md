# HTMX enhancement

Northframe ships a pinned copy of HTMX for applications that want HTML-over-the-wire interactions. It is optional: server-rendered pages, links, and forms remain ordinary HTTP first. HTMX adds partial navigation, polling, loading indicators, targeted swaps, and event-driven refreshes without introducing a JavaScript server or an application `package.json`.

Northframe currently embeds HTMX 2.0.10 at `/_northframe/htmx.min.js`. The asset is compiled into the Go deployment binary and its license is recorded in the repository's third-party notices.

## Enable HTMX in the root layout

Load the embedded asset once in `web/routes/layout.north`:

```html
<head>
  <link rel="stylesheet" href="/_northframe/app.css" />
  <script src="/_northframe/htmx.min.js" defer></script>
</head>
```

Northframe does not load HTMX automatically. An application that does not include this script ships no HTMX browser code.

## Enhance a native form

Begin with a form that has a working `method` and `action`, then add HTMX attributes:

```html
<form
  method="post"
  action="/jobs"
  hx-post="/jobs"
  hx-target="#dispatch-board"
  hx-select="#dispatch-board"
  hx-swap="outerHTML"
>
  <input name="destination" required />
  <button type="submit">Create job</button>
</form>
```

Without JavaScript, the browser submits `POST /jobs` normally. With HTMX, the same endpoint is requested in the background. If the action returns a `303` redirect to the page, HTMX follows it, selects `#dispatch-board` from the rendered response, and replaces the current board.

`hx-target` selects the element to replace. `hx-select` selects the matching fragment from a full HTML response. This pattern lets a Northframe loader remain the single rendering path while HTMX performs a partial update.

## Detect HTMX in Go

Page actions receive the normal `*web.Context`:

```go
func createJob(ctx *web.Context) error {
    queue := web.MustUse[*Queue](ctx)
    if err := queue.Create(ctx.StdContext(), ctx.FormValue("destination")); err != nil {
        return err
    }

    if ctx.HTMX() {
        _ = ctx.HXTriggerEventsAfterSwap(map[string]any{
            "job-created": map[string]string{"source": "dispatch"},
        })
    }
    return ctx.Redirect("/jobs", http.StatusSeeOther)
}
```

`ctx.HTMX()` reads `HX-Request`. The request helpers also expose boosted navigation, the current URL, prompt value, target ID, and triggering control name:

```go
ctx.HXBoosted()
ctx.HXCurrentURL()
ctx.HXPrompt()
ctx.HXTarget()
ctx.HXTriggerName()
```

## Control the HTMX response

Northframe provides typed names for the standard response headers:

```go
ctx.HXLocation("/jobs/active")
ctx.HXRedirect("/login")
ctx.HXRefresh()
ctx.HXPushURL("/jobs?page=2")
ctx.HXReplaceURL("/jobs")
ctx.HXRetarget("#form-errors")
ctx.HXReselect("#updated-row")
ctx.HXReswap("outerHTML")
ctx.HXTrigger("job-created")
ctx.HXTriggerEvents(map[string]any{"job-created": map[string]string{"id": "JOB-42"}})
```

These methods set response headers; they do not write a response body. Call them from actions and API handlers where `ctx.Response` is available, then return HTML, a redirect, or no content as the interaction requires. A page or layout loader cannot set response headers because loaders render into an intermediate buffer.

## Poll a server-rendered region

HTMX polling is useful for dashboards whose source of truth remains in Go:

```html
<section
  id="fleet-state"
  hx-get="/operations"
  hx-trigger="every 3s"
  hx-target="#fleet-state"
  hx-select="#fleet-state"
  hx-swap="outerHTML"
>
  <!-- rendered from current Go props -->
</section>
```

The browser asks for the page every three seconds but swaps only the selected region. The loader still performs authorization and reads the current service state. Stop or lengthen polling when the tab is hidden for high-volume applications.

## HTMX versus component TypeScript

Use HTMX for interactions where the server owns the durable state and can respond with HTML: forms, filters, pagination, queue operations, server-side search, polling, and partial navigation.

Use a colocated `<script lang="ts">` for browser-only state: an open dialog, a map zoom level, keyboard controls, drag-and-drop, a chart cursor, or a complex editor. HTMX does not replace TypeScript islands, and TypeScript should not duplicate authoritative database or workflow state.

## Security and testing

HTMX requests are ordinary HTTP requests. Continue to enforce authentication, authorization, CSRF protection, validation, body limits, and escaping on the server. Never treat `HX-Request` as proof that a request came from trusted code.

Test the native form path first. Then add an HTTP test with `HX-Request: true` and assert any response headers. Keep a small number of browser tests for the actual swap, history, focus, and loading behavior.
