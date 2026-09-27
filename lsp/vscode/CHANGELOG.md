# Changelog

## 0.1.0 (Pre-release)

First public beta of the Northframe VS Code extension.

- Register `.north` with the Northframe language server and selected flame icon.
- Highlight typed `Props` frontmatter, Go server expressions and control blocks,
  TypeScript client state, HTML, and component syntax.
- Provide compiler diagnostics, completion, hover, Go to Definition, Find All
  References, rename, document symbols, and snippets.
- Navigate through generated route props, imported Go structs, loop values,
  component tags, and individual component prop attributes.
- Delegate embedded HTML, TypeScript, Emmet, and Tailwind intelligence through
  position-preserving virtual documents.
- Complete project, standard-library, and `go.mod` dependency packages in
  `interface Props` and add missing imports while completing Go types.
- Resolve and sort unambiguous Go imports automatically on save without changing
  `go.mod`.
- Format HTML, Go-shaped directives, Props fields, and TypeScript independently.
- Keep multiline `>` and `/>` brackets with the final attribute and preserve
  embedded TypeScript indentation.
- Reject formatting edits that delete or duplicate meaningful Northframe tokens.
- Prefer the open Northframe source checkout over a stale global CLI while
  developing the framework or an included example.
