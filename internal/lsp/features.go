package lsp

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

var symbolPattern = regexp.MustCompile(`(?i)<(main|nav|section|form|h[1-6])(?:\s[^>]*)?>`)

func (current *server) hover(id json.RawMessage, raw json.RawMessage) error {
	var params textDocumentPosition
	if err := json.Unmarshal(raw, &params); err != nil {
		return err
	}
	text := current.document(params.TextDocument.URI)
	offset := positionToByteOffset(text, params.Position)
	word := wordAt(text, offset)
	propWord := tokenRoot(word)
	if strings.HasPrefix(word, "Props.") {
		propWord = tokenRoot(strings.TrimPrefix(word, "Props."))
	}
	if symbol, ok := importedGoTypeAt(params.TextDocument.URI, text, offset); ok {
		value := "### Imported Go type\n\n```go\n" + symbol.Declaration + "\n```"
		if symbol.Documentation != "" {
			value += "\n\n" + symbol.Documentation
		}
		value += "\n\nUse **Go to Definition** to open the declaring Go file."
		return current.reply(id, map[string]any{"contents": map[string]string{"kind": "markdown", "value": value}})
	}
	if value, ok := templateValueAt(params.TextDocument.URI, text, offset); ok {
		declaration := value.Name + " " + value.Type
		markdown := "### Typed Go value\n\n```go\n" + declaration + "\n```"
		if value.Owner != "" {
			markdown += "\n\nDeclared on `" + value.Owner + "`."
		}
		if value.Documentation != "" {
			markdown += "\n\n" + value.Documentation
		}
		markdown += "\n\nUse **Go to Definition** to open its declaration."
		return current.reply(id, map[string]any{
			"contents": map[string]string{"kind": "markdown", "value": markdown},
			"range":    wordRange(text, offset),
		})
	}
	if field, ok := findDocumentProp(params.TextDocument.URI, text, propWord); ok {
		return current.reply(id, map[string]any{
			"contents": map[string]string{"kind": "markdown", "value": "### Server property\n\n```go\n" + field.Name + " " + field.Type + "\n```\n\nA type-safe value supplied by the route's Go loader and rendered during SSR. Reference it in markup as `{Props." + field.Name + "}`."},
		})
	}
	if found, ok := findComponent(params.TextDocument.URI, word); ok {
		props := "This component has no required props."
		if len(found.Props) > 0 {
			quoted := make([]string, 0, len(found.Props))
			for _, prop := range found.Props {
				quoted = append(quoted, "`"+prop+"`")
			}
			props = "**Props:** " + strings.Join(quoted, ", ")
		}
		children := ""
		if found.HasSlot {
			children = "\n\nAccepts child markup through `<slot />`."
		}
		events := ""
		if len(found.Events) > 0 {
			quoted := make([]string, 0, len(found.Events))
			for _, event := range found.Events {
				quoted = append(quoted, "`on:"+event+"`")
			}
			events = "\n\n**Events:** " + strings.Join(quoted, ", ")
		}
		return current.reply(id, map[string]any{
			"contents": map[string]string{"kind": "markdown", "value": "### `<" + found.Name + ">`\n\nIndependent Northframe component with type-checked Go props and isolated browser state.\n\n" + props + children + events + "\n\nUse **Go to Definition** to open its `.north` source."},
		})
	}
	if help, ok := hoverHelpAt(text, offset); ok {
		return current.reply(id, map[string]any{"contents": map[string]string{"kind": "markdown", "value": help}})
	}
	return current.reply(id, nil)
}

type hoverTopic struct {
	pattern  *regexp.Regexp
	markdown string
}

var hoverTopics = []hoverTopic{
	{
		regexp.MustCompile(`interface\s+Props\s*\{`),
		"### Typed server Props\n\nAstro-style `---` frontmatter is a declarative Go type contract. Put Go imports first, then declare `interface Props`; Northframe generates `PageProps`, `LayoutProps`, or component props.\n\n```north\n---\nimport uuid \"github.com/google/uuid\"\nimport viewmodels \"example/internal/viewmodels\"\n\ninterface Props {\n  ID uuid.UUID\n  Items []viewmodels.Item\n}\n---\n```\n\nStandard-library, `go.mod` dependency, and project types support automatic imports, completion, hover, and Go to Definition. Executable Go functions stay in `.north.go` loaders or ordinary Go packages. Use prop values in markup as `{Props.ID}`. Browser state remains in Svelte-style `<script lang=\"ts\">`.",
	},
	{
		regexp.MustCompile(`<script\b[^>]*\blang\s*=\s*["']ts["'][^>]*>`),
		"### Browser TypeScript\n\n`<script lang=\"ts\">` declares page- or component-local client state. Northframe type-checks and compiles it to JavaScript—no WebAssembly is required.\n\nReference variables with `{#name}` and attach functions with attributes such as `on:click={handler}`.",
	},
	{
		regexp.MustCompile(`\{for\b[^}]*\}`),
		"### Go server loop\n\n`{for item := range Props.Items}` iterates a typed Go slice or array during SSR. Northframe emits `for _, item := range props.Items` and formats the renderer with `gofmt`.\n\n```north\n{for item := range Props.Items}\n  <InventoryCard Item={item} />\n{/for}\n```",
	},
	{
		regexp.MustCompile(`\{/for\}`),
		"### End Go server loop\n\n`{/for}` closes the nearest `{for ... := range ...}` block.",
	},
	{
		regexp.MustCompile(`\{if\b[^}]*\}`),
		"### Go server condition\n\n`{if Props.Condition}` renders its body during SSR when the value is truthy. Full Go boolean expressions such as `{if len(Props.Items) > 0}` compile directly. Close it with `{/if}`.",
	},
	{
		regexp.MustCompile(`\{html\b[^}]*\}`),
		"### Sanitized server HTML\n\n`{html Props.Content}` renders a typed `web.SafeHTML` value without escaping it again. Ordinary strings are rejected by Go's type checker. Create the value only after an allow-list sanitizer has removed scripts, event handlers, and unsafe URLs:\n\n```go\nContent: web.SafeHTMLFromSanitized(sanitize.RichText(value))\n```",
	},
	{
		regexp.MustCompile(`\{/if\}`),
		"### End server condition\n\n`{/if}` closes the nearest `{if ...}` block.",
	},
	{
		regexp.MustCompile(`\{#include\b[^}]*\}`),
		"### Compile-time include\n\n`{#include path/name}` inserts a static component source while compiling. Prefer an independent PascalCase component when you need typed props or isolated state.",
	},
	{
		regexp.MustCompile(`<slot\b[^>]*>`),
		"### Component slot\n\n`<slot />` renders child markup passed by a layout or parent component as a native Go fragment.",
	},
	{
		regexp.MustCompile(`\{\s*(?:Props\.)?[A-Za-z_][A-Za-z0-9_.]*\s*\}`),
		"### Go server expression\n\n`{Props.Value}` escapes and renders typed Go data during SSR. Loop locals use normal Go names, for example `{item.Name}`.",
	},
	{
		regexp.MustCompile(`\{#[A-Za-z_][A-Za-z0-9_.]*\}`),
		"### Client expression\n\n`{#value}` renders reactive TypeScript state in the browser. `#` always means client-owned state; Northframe compiles it to JavaScript without WebAssembly.",
	},
	{
		regexp.MustCompile(`on:[A-Za-z][A-Za-z0-9_-]*`),
		"### Client event\n\n`on:event={handler}` runs a function or expression from `<script lang=\"ts\">`, then refreshes Northframe client bindings. It also listens to bubbling events emitted by an independent component with `dispatch(\"event\", detail)`.",
	},
	{
		regexp.MustCompile(`\bdispatch\s*\(`),
		"### Component event dispatch\n\n`dispatch(\"event\", detail)` emits a bubbling `CustomEvent` from the current independent component. A parent listens with `on:event={handler}` and receives the typed value on `event.detail`.",
	},
	{
		regexp.MustCompile(`bind:value`),
		"### Two-way value binding\n\n`bind:value={#name}` keeps an input's value and a TypeScript state variable synchronized.",
	},
	{
		regexp.MustCompile(`class:[A-Za-z_][A-Za-z0-9_-]*`),
		"### Reactive class\n\n`class:name={#enabled}` adds or removes one CSS class from a client-state boolean.",
	},
	{
		regexp.MustCompile(`\bshow\s*=`),
		"### Reactive visibility\n\n`show={#open}` toggles the element's native `hidden` state from a TypeScript value.",
	},
	{
		regexp.MustCompile(`nf-enhance`),
		"### Enhanced form\n\n`nf-enhance` submits a native form in the background, updates pending/error UI, and preserves the ordinary no-JavaScript POST fallback.",
	},
	{
		regexp.MustCompile(`nf-loading`),
		"### Pending state\n\nAn element marked `nf-loading` is shown while its nearest enhanced form request is running. Add `hidden` so SSR starts in the idle state.",
	},
	{
		regexp.MustCompile(`nf-idle`),
		"### Idle state\n\nAn element marked `nf-idle` is visible while its nearest enhanced form is not submitting.",
	},
	{
		regexp.MustCompile(`nf-error`),
		"### Error state\n\nAn element marked `nf-error` is shown when an enhanced request fails.",
	},
	{
		regexp.MustCompile(`nf-message`),
		"### Response message\n\nAn element marked `nf-message` receives the status message returned by an enhanced form action.",
	},
}

func hoverHelpAt(text string, offset int) (string, bool) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(text) {
		offset = len(text)
	}
	lineStart := strings.LastIndex(text[:offset], "\n") + 1
	lineEnd := strings.Index(text[offset:], "\n")
	if lineEnd < 0 {
		lineEnd = len(text)
	} else {
		lineEnd += offset
	}
	line := text[lineStart:lineEnd]
	cursor := offset - lineStart
	for _, topic := range hoverTopics {
		for _, match := range topic.pattern.FindAllStringIndex(line, -1) {
			if cursor >= match[0] && cursor <= match[1] {
				return topic.markdown, true
			}
		}
	}
	return "", false
}

func (current *server) completion(id json.RawMessage, raw json.RawMessage) error {
	var params textDocumentPosition
	if err := json.Unmarshal(raw, &params); err != nil {
		return err
	}
	items := []map[string]any{
		{"label": "Props frontmatter", "kind": 15, "insertText": "---\n${1:import models \"example/internal/models\"}\n\ninterface Props {\n\t${2:Title} ${3:string}\n}\n---\n", "insertTextFormat": 2, "detail": "Go imports and typed server props"},
		{"label": "{Props.Value}", "kind": 15, "insertText": "{Props.${1:Value}}", "insertTextFormat": 2, "detail": "Escaped Go SSR value"},
		{"label": "{html}", "kind": 15, "insertText": "{html Props.${1:ContentHTML}}", "insertTextFormat": 2, "detail": "Render a typed web.SafeHTML value"},
		{"label": "{#state}", "kind": 15, "insertText": "{#${1:name}}", "insertTextFormat": 2, "detail": "Reactive TypeScript value"},
		{"label": "{if}", "kind": 15, "insertText": "{if Props.${1:Condition}}\n\t$0\n{/if}", "insertTextFormat": 2, "detail": "Go server condition"},
		{"label": "{for}", "kind": 15, "insertText": "{for ${1:item} := range Props.${2:Items}}\n\t$0\n{/for}", "insertTextFormat": 2, "detail": "Go server range loop"},
		{"label": "{#include}", "kind": 15, "insertText": "{#include $0}", "insertTextFormat": 2, "detail": "Static component include"},
		{"label": "<Component />", "kind": 7, "insertText": "<${1:Component} ${2:Prop}={Props.${3:Value}} />", "insertTextFormat": 2, "detail": "Independent typed component"},
		{"label": "<slot />", "kind": 15, "insertText": "<slot />", "detail": "Layout content slot"},
		{"label": "<script lang=\"ts\">", "kind": 15, "insertText": "<script lang=\"ts\">\nlet ${1:open}: boolean = false;\n</script>", "insertTextFormat": 2, "detail": "Compiled route TypeScript"},
		{"label": "on:click", "kind": 10, "insertText": "on:click={${1:handler}}", "insertTextFormat": 2, "detail": "TypeScript event handler"},
		{"label": "dispatch", "kind": 3, "insertText": "dispatch(\"${1:event}\", ${2:detail});", "insertTextFormat": 2, "detail": "Emit a bubbling component CustomEvent"},
		{"label": "show", "kind": 10, "insertText": "show={#${1:open}}", "insertTextFormat": 2, "detail": "Reactive visibility binding"},
		{"label": "bind:value", "kind": 10, "insertText": "bind:value={#${1:value}}", "insertTextFormat": 2, "detail": "Two-way TypeScript state binding"},
		{"label": "class:name", "kind": 10, "insertText": "class:${1:active}={#${2:enabled}}", "insertTextFormat": 2, "detail": "Reactive class binding"},
		{"label": "nf-enhance", "kind": 10, "insertText": "nf-enhance", "detail": "Progressively enhance a POST form"},
		{"label": "nf-loading", "kind": 10, "insertText": "nf-loading hidden", "detail": "Visible while an enhanced form is pending"},
		{"label": "nf-idle", "kind": 10, "insertText": "nf-idle", "detail": "Visible while an enhanced form is idle"},
	}
	text := current.document(params.TextDocument.URI)
	items = append(items, propsImportCompletionItems(params.TextDocument.URI, text, positionToByteOffset(text, params.Position))...)
	items = append(items, goTypeCompletionItems(params.TextDocument.URI, text, positionToByteOffset(text, params.Position))...)
	items = append(items, templateFieldCompletionItems(params.TextDocument.URI, text, positionToByteOffset(text, params.Position))...)
	items = append(items, clientImportCompletionItems(params.TextDocument.URI, text, positionToByteOffset(text, params.Position))...)
	for _, field := range documentProps(params.TextDocument.URI, text) {
		items = append(items, map[string]any{"label": field.Name, "kind": 5, "detail": field.Type + " — typed route property"})
	}
	for _, component := range projectComponents(params.TextDocument.URI) {
		var attributes []string
		for index, prop := range component.Props {
			attributes = append(attributes, prop+"={Props.${"+strconv.Itoa(index+1)+":Value}}")
		}
		ending := " />"
		if component.HasSlot {
			ending = ">$0</" + component.Name + ">"
		}
		snippet := "<" + component.Name
		if len(attributes) > 0 {
			snippet += " " + strings.Join(attributes, " ")
		}
		snippet += ending
		items = append(items, map[string]any{"label": "<" + component.Name + ">", "kind": 7, "insertText": snippet, "insertTextFormat": 2, "detail": "Independent typed component"})
		for _, event := range component.Events {
			items = append(items, map[string]any{"label": "on:" + event + " (" + component.Name + ")", "kind": 10, "insertText": "on:" + event + "={${1:handler}}", "insertTextFormat": 2, "detail": "Event emitted by <" + component.Name + ">"})
		}
	}
	return current.reply(id, map[string]any{"isIncomplete": false, "items": items})
}

func (current *server) definition(id json.RawMessage, raw json.RawMessage) error {
	var params textDocumentPosition
	if err := json.Unmarshal(raw, &params); err != nil {
		return err
	}
	text := current.document(params.TextDocument.URI)
	offset := positionToByteOffset(text, params.Position)
	if symbol, ok := importedGoTypeAt(params.TextDocument.URI, text, offset); ok {
		return current.reply(id, map[string]any{"uri": symbol.URI, "range": symbol.Range})
	}
	if value, ok := templateValueAt(params.TextDocument.URI, text, offset); ok && value.URI != "" {
		return current.reply(id, map[string]any{"uri": value.URI, "range": value.Range})
	}
	word := wordAt(text, offset)
	if strings.HasPrefix(word, "Props.") {
		word = strings.TrimPrefix(word, "Props.")
	}
	word = tokenRoot(word)
	field, ok := findDocumentProp(params.TextDocument.URI, text, word)
	if ok {
		return current.reply(id, map[string]any{"uri": field.URI, "range": field.Range})
	}
	if component, exists := findComponent(params.TextDocument.URI, word); exists {
		return current.reply(id, map[string]any{"uri": documentURI(component.Path), "range": protocolRange{Start: position{}, End: position{Character: 1}}})
	}
	return current.reply(id, nil)
}

func (current *server) documentSymbols(id json.RawMessage, raw json.RawMessage) error {
	var params struct {
		TextDocument versionedTextDocument `json:"textDocument"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return err
	}
	text := current.document(params.TextDocument.URI)
	var symbols []map[string]any
	for _, match := range symbolPattern.FindAllStringSubmatchIndex(text, -1) {
		start := byteOffsetToPosition(text, match[0])
		end := byteOffsetToPosition(text, match[1])
		name := strings.ToLower(text[match[2]:match[3]])
		symbols = append(symbols, map[string]any{
			"name": name, "kind": 8,
			"range":          protocolRange{Start: start, End: end},
			"selectionRange": protocolRange{Start: start, End: end},
		})
	}
	return current.reply(id, symbols)
}

func (current *server) formatting(id json.RawMessage, raw json.RawMessage) error {
	var params struct {
		TextDocument versionedTextDocument `json:"textDocument"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return err
	}
	text := current.document(params.TextDocument.URI)
	formatted := formatDocument(text)
	if formatted == text {
		return current.reply(id, []any{})
	}
	return current.reply(id, []map[string]any{{"range": fullDocumentRange(text), "newText": formatted}})
}

func formatDocument(text string) string {
	lines := strings.Split(strings.ReplaceAll(text, "<slot/>", "<slot />"), "\n")
	for index := range lines {
		lines[index] = strings.TrimRight(lines[index], " \t")
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
}

func lineAt(text string, offset int) string {
	start := strings.LastIndex(text[:offset], "\n") + 1
	endOffset := strings.Index(text[offset:], "\n")
	if endOffset < 0 {
		return text[start:]
	}
	return text[start : offset+endOffset]
}
