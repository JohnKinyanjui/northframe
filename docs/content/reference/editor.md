# Editor and language server

Northframe ships one language server with the CLI. It uses the production compiler parser, understands the Go-facing and browser-facing regions of a `.north` file, and coordinates VS Code's HTML and TypeScript services without adding editor tooling to an application.

Run the server directly only when integrating another editor:

```sh
north lsp
```

The command speaks LSP over standard input and output; it is not an interactive shell command. The VS Code extension starts it automatically.

## Install the beta VS Code extension

Install the matching CLI first:

```sh
go install github.com/JohnKinyanjui/northframe/cmd/cli@v0.1.0-beta
```

Until the extension is available from a registry, package it from a Northframe checkout:

```sh
cd lsp/vscode
npx @vscode/vsce package --out northframe.vsix
code --install-extension "$PWD/northframe.vsix" --force
```

Run **Developer: Reload Window** after installation. Open a `.north` file and use **Northframe: Show Language Server Output** to confirm which executable was started. The extension requires VS Code's built-in HTML and TypeScript features and installs Tailwind CSS IntelliSense as an editor dependency.

The Node command above packages a VS Code extension. It does not add Node, npm, or `package.json` to a Northframe application.

## What the language server understands

Northframe does not treat the whole file as approximate HTML. It understands each language region separately:

- Go imports and `interface Props` inside frontmatter.
- `${...}`, `{if ...}`, and `{for ...}` server expressions.
- Component tags, typed attributes, slots, and emitted events.
- TypeScript declarations and imports inside `<script lang="ts">`.
- `#{...}` client bindings and `on:event` handlers.
- HTML attributes, accessibility information, and Tailwind classes.

Hovering a prop shows its Go type and declaration owner. Go to Definition on `Props.Products` opens the frontmatter or sidecar declaration. Inside a loop, `product.Name` resolves through the collection element's imported Go struct. Go to Definition on `<CatalogProductRow>` opens its `.north` source, while using it on the `Product=` attribute opens that component's prop declaration. Find All References works for props, loop variables, component tags, and component prop attributes.

TypeScript navigation is delegated to VS Code's TypeScript language service through a position-preserving virtual document. HTML, Emmet, and Tailwind completion use the same technique, so their locations map back to the original `.north` file.

The language server uses the same parser as the compiler. This means a diagnostic in the editor should be actionable at build time too. If a CLI upgrade changes compiler behavior, restart the language server and the `north run` process so they load the new executable.

## Save-time Go imports

The frontmatter region is a Go type contract, so its packages should behave like Go imports. You may type a qualified field before adding the import:

```north
---
interface Props {
  ID uuid.UUID
  CreatedAt time.Time
}
---
```

When you save, the extension asks `north lsp` to resolve package aliases. It searches:

- packages in the current Go module;
- standard-library packages;
- dependencies already present in `go.mod`;
- replace targets and source available through the Go module graph.

The result is added above `interface Props` and sorted by path:

```north
---
import uuid "github.com/google/uuid"
import time "time"

interface Props {
  ID uuid.UUID
  CreatedAt time.Time
}
---
```

Northframe does not run `go get` during save. That would make an editor keystroke change the dependency graph. Add a new dependency explicitly, then save the `.north` file. When an alias remains ambiguous after preferring the current module and direct dependencies, the language server leaves it unresolved and reports a diagnostic rather than importing the wrong package.

## Formatting behavior

The formatter processes every language region separately:

- Props fields align like Go declarations and imports are sorted.
- `<script lang="ts">` uses the TypeScript formatter without flattening function bodies.
- HTML children, attributes, and closing tags receive stable indentation.
- `{if ...}` and `{for item := range ...}` retain Go-shaped spacing and nesting.
- server and client interpolation tokens remain unchanged.

Long opening tags place one attribute on each line while keeping the final bracket with the last attribute:

```north
<button
  type="button"
  aria-label="Close dialog"
  class="h-9 w-9 rounded-full">
  Close
</button>

<SearchField
  Label="Search"
  Placeholder="Search orders, products, or customers" />
```

Before applying the edit, Northframe compares the meaningful token stream before and after formatting. If a delegated formatter deletes a function token, duplicates markup, or damages an interpolation, the edit is rejected and the reason is written to the Northframe output channel.

Format-on-save is configured for the `northframe` language by the extension. Save-time import organization also runs independently, so missing Go imports are added even when you manually invoke save without running a separate formatting command.

## Components and imports

Northframe components do not need manual template imports. A file such as `web/components/catalog/ProductRow.north` is discovered as `<CatalogProductRow>`. Completion inserts its typed props, Go to Definition opens its source, and Find All References finds callers.

Imports inside `<script lang="ts">` are browser packages, not Go packages. Packages declared in `northframe.toml` and local modules under `web/client` are offered by TypeScript completion. Imports in `---` are only for Go types used by `Props` and Go defaults.

The sibling `page.north.go` or `layout.north.go` remains ordinary Go. Use the official Go extension and gopls there. Northframe does not proxy Go diagnostics or format those files.

## Keep the CLI and extension aligned

The extension launches the executable configured by `northframe.server.path`, which defaults to `north`. Check the exact binary before reporting a parser problem:

```sh
command -v north
go install github.com/JohnKinyanjui/northframe/cmd/cli@latest
```

After replacing the CLI, run **Northframe: Restart Language Server** from the command palette. An already-running extension process does not silently switch to the new executable.

If one terminal uses a local development binary while VS Code uses `~/go/bin/north`, they can disagree about recently added syntax. Open **Northframe: Show Language Server Output** to see the command the extension started. Set `northframe.server.path` to an absolute path when deliberately testing a local compiler.

## Navigation expectations

- Clicking an imported `uuid.UUID` type opens the declaring Go package when source is available.
- Clicking `Props.Page` opens the matching typed prop declaration.
- Clicking a loop field such as `item.Name` opens the Go struct field.
- Clicking `<DocsPage>` opens `web/components/DocsPage.north`.
- Clicking `Page=` on that invocation opens `DocsPage`'s `interface Props` field.
- Clicking a TypeScript function or imported browser symbol uses the TypeScript provider.

If navigation returns the wrong symbol, place the cursor directly on the identifier rather than `${`, `=`, or an angle bracket. Save unsaved component definitions before asking for project-wide references because unopened files are read from disk.

## Troubleshooting

When diagnostics look stale, save the file, restart the Northframe language server, and confirm the configured command resolves to the same `north` executable used in your terminal. Run `north generate` from the application root; it uses the production parser and is the final authority.

If only TypeScript navigation is missing, confirm VS Code's built-in TypeScript and JavaScript language features are enabled. If only Tailwind completion is missing, check that the Tailwind CSS extension is enabled and that `tailwindCSS.includeLanguages` still maps `northframe` to `html`.

If save does not add a Go import, check that the type is inside `interface Props`, uses a qualifier such as `uuid.UUID`, and belongs to the current module, standard library, or an existing `go.mod` dependency. Open the Northframe output channel to verify which CLI is running.

If formatting still leaves a detached bracket, confirm the active formatter for `.north` is **Northframe**, then run **Developer: Reload Window**. **Format Document With...** lets you select Northframe as the default if another HTML formatter took ownership.
