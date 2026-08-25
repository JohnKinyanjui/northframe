# Changelog

## Development

- Replace Svelte-style server blocks with Go-shaped `{if ...}` and `{for item := range ...}` syntax.
- Highlight server expressions with the Go grammar and normalize directive spacing during formatting.
- Move Go package imports before `interface Props` inside Astro-style frontmatter.
- Complete exported project Go types after an imported package qualifier.
- Add hover documentation and Go-to-definition for imported structs, interfaces, and named types.
- Follow `Props` and `{for value := range ...}` variables into nested Go struct fields for completion, hover, and go-to-definition.
- Offer missing project package imports while completing Go types in `interface Props`.
- Resolve standard-library and `go.mod` dependency types such as `uuid.UUID` through Go's package index.
- Adopt the `web/routes`, `web/components`, `web/client`, and `web/public` project convention.
- Keep the full frontmatter block protected while HTML formatting runs.

## 0.2.5

- Prevent overlapping HTML formatter edits from duplicating or corrupting `.north` component tails.
- Add regression coverage for formatter idempotence on long interactive components.

## 0.2.4

- Color `context="props"` and `lang="ts"` as real tag attributes and values.
- Add rich, cursor-aware hover documentation for server props, browser state, control blocks, components, bindings, and enhanced-form states.

## 0.2.3

- Give props contracts dedicated Northframe scopes instead of treating the DSL as invalid Go.
- Highlight complete `{#each}`, `{#if}`, and component-tag expressions consistently.
- Indent props contracts and server control-block bodies during document formatting.

## 0.2.2

- Adopt the selected Northframe flame mascot and its high-contrast VS Code adaptation.

## 0.2.1

- Add the Northframe fire-and-code marketplace icon.

## 0.2.0

- Forward embedded HTML completion and hover to VS Code's HTML language service.
- Forward `<script lang="ts">` completion, hover, and formatting to the TypeScript language service.
- Complete project-aware Go imports in props blocks from the nearest `go.mod`.
- Format `.north` documents as HTML while preserving and independently formatting embedded scripts and props blocks.
- Map Northframe to HTML for Emmet and Tailwind CSS IntelliSense.

## 0.1.0

- Register `.north` as the Northframe language.
- Launch the built-in `north lsp` server over stdio.
- Add diagnostics, completion, hover, definitions, symbols, and formatting.
- Add Northframe syntax highlighting, language configuration, and snippets.
