# Northframe for VS Code

Language support for `.north` files backed by Northframe's built-in Go language
server.

## Features

- compiler diagnostics
- typed `Props` completion for pages, layouts, and independent components
- independent component completion
- hover and go-to-definition
- document symbols and formatting
- Northframe, HTML, TypeScript, and `Props` frontmatter highlighting
- HTML tag, attribute, hover, and formatting support from VS Code's HTML service
- TypeScript completion, hover, and formatting inside `script lang="ts"`
- `northframe.toml` package and `$client/...` import completion
- Go-like imports before `interface Props`, with project package completion
- exported Go struct/interface completion with automatic imports from the standard
  library, project packages, and `go.mod` dependencies
- automatic Go import resolution and path sorting whenever a `.north` file is saved
- Go to Definition and Find All References for typed props, loop bindings,
  component tags, and individual component prop attributes
- native TypeScript definition/reference navigation inside `<script lang="ts">`
- field completion, rich hover, and go-to-definition through `Props`, loop variables,
  and nested Go structs
- Tailwind CSS IntelliSense and Emmet through Northframe-to-HTML language mapping
- snippets for state, server blocks, and enhanced forms

The formatter keeps `>` and `/>` beside the final attribute of a multiline tag,
preserves embedded TypeScript indentation, and rejects any delegated edit that
deletes or duplicates meaningful Northframe tokens.

## Requirements

Install the Northframe CLI so `north` is available on `PATH`, then open a
Northframe project and any `.north` file. The extension starts `north lsp`
automatically.

The default project convention keeps browser-facing code together under
`web/routes`, `web/components`, `web/client`, and `web/public`. Go packages
elsewhere in the project remain available to `Props` imports and navigation.

The Tailwind CSS IntelliSense extension is installed as an editor dependency.
Northframe maps `.north` to its HTML mode; no application `package.json` is
introduced by this integration.

If the CLI is elsewhere, configure:

```json
{
  "northframe.server.path": "/absolute/path/to/north"
}
```

Use **Northframe: Restart Language Server** after changing the executable.

## Development installation

No dependency installation or JavaScript build is required. From this
directory, run:

```sh
code --extensionDevelopmentPath="$PWD"
```

The `package.json` in this directory is the VS Code extension manifest. It does
not add a `package.json` requirement to Northframe applications.
