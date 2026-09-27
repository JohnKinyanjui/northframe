# Installation

## Requirements

- Go 1.27 or newer
- A Go module for your application

Node, Deno, npm, and `package.json` are not required for an application. Northframe downloads browser dependencies into its own local store when you use `northframe add`.

## Install the CLI

For application development, install the module with Go:

```sh
go install github.com/JohnKinyanjui/northframe/cmd/northframe@v0.1.0-beta
```

That is the recommended beta installation because it is reproducible. Use `@latest` only when you deliberately want the newest published Northframe release. You do not need to clone Northframe, manually copy a binary, or install a JavaScript package manager. The source is available in the [Northframe GitHub repository](https://github.com/JohnKinyanjui/northframe).

Verify that Go’s install directory is on your `PATH`:

```sh
command -v northframe
northframe help
```

If `command -v northframe` prints a path, installation is complete. `northframe help` should list commands such as `create`, `run`, `build`, `generate`, and `upgrade`.

### Install from a source checkout

Contributors working on Northframe itself can install the local checkout:

```sh
go install ./cmd/northframe
```

Make sure the directory reported by `go env GOBIN` (or `$(go env GOPATH)/bin` when `GOBIN` is empty) is on your `PATH`, then verify:

```sh
northframe help
```

The CLI is intentionally installed separately from an application. Updating the executable does not rewrite route files or application code. Do not use the checkout command when you intended to install the published module.

## Install the VS Code extension

Install the beta extension directly from the [Visual Studio Marketplace](https://marketplace.visualstudio.com/items?itemName=JohnKinyanjui.northframe). In VS Code, search for **Northframe**, open the extension, use the Install button's dropdown, and choose **Install Pre-Release Version**. From a terminal you can use:

```sh
code --install-extension JohnKinyanjui.northframe --pre-release
```

The extension starts `northframe lsp` automatically. Keep the CLI and extension from the same release family; after replacing the CLI, run **Northframe: Restart Language Server**.

Contributors can still package and install the extension from a Northframe checkout:

```sh
cd lsp/vscode
npx @vscode/vsce package --out northframe.vsix
code --install-extension "$PWD/northframe.vsix" --force
```

This local packaging step uses Node because VS Code extensions are JavaScript packages. Marketplace users and Northframe applications do not need Node, npm, or an application `package.json`. After installation, run **Developer: Reload Window**, open a `.north` file, and check **Northframe: Show Language Server Output** if the server does not start.

## Create an application

Create a new directory and scaffold it:

```sh
mkdir hello-north
cd hello-north
northframe create .
northframe run
```

`northframe create .` performs a conflict preflight before writing. It will not overwrite a non-matching file already in the target directory. Open <http://localhost:8000> to view the generated page.

The scaffold includes `go.mod`, `northframe.toml`, a root layout, a root page, a health route, and ignored generated output. Replace the starter markup after confirming the server works.

### Create a documentation site

Documentation is a native Northframe project type. It uses the same router, `.north` components, TypeScript islands, error handling, and single-binary deployment as an application, while `pkg/docs` safely renders application-owned Markdown:

```sh
mkdir product-docs
cd product-docs
northframe create . --template=docs
northframe run
```

The docs template creates `content/` for Markdown, a filtered sidebar and responsive documentation shell under `web/`, and an embedded content loader. This is not an Astro or Node project: editing Markdown and `.north` files is enough.

## Updating safely

After installing a newer CLI, preview the framework-managed refresh with `northframe upgrade --check`, then apply it with `northframe upgrade`. Restart an existing `northframe run` process after replacing the executable because the watcher keeps the compiler loaded in memory.

If `northframe` is not found, add `$(go env GOPATH)/bin` to `PATH` (or the directory printed by `go env GOBIN`). If `northframe create .` reports conflicts, inspect the listed files instead of deleting them; scaffolding refuses to overwrite existing application files.
