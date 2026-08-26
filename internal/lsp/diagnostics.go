package lsp

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"northframe.dev/northframe/internal/compiler"
)

var bytePosition = regexp.MustCompile(`byte ([0-9]+)`)
var serverHTMLExpression = regexp.MustCompile(`\{html\s+([^}]+)\}`)

func (current *server) publishDiagnostics(uri, text string) error {
	diagnostics := validateDocument(uri, text)
	return current.transport.write(notification{
		JSONRPC: "2.0", Method: "textDocument/publishDiagnostics",
		Params: map[string]any{"uri": uri, "diagnostics": diagnostics},
	})
}

func validateDocument(uri, text string) []diagnostic {
	var diagnostics []diagnostic
	if err := compiler.Validate([]byte(text)); err != nil {
		offset := validationErrorOffset(err.Error())
		start := byteOffsetToPosition(text, offset)
		diagnostics = append(diagnostics, diagnostic{
			Range:    protocolRange{Start: start, End: byteOffsetToPosition(text, min(offset+1, len(text)))},
			Severity: 1, Source: "northframe", Message: err.Error(),
		})
	}
	diagnostics = append(diagnostics, propsTypeDiagnostics(uri, text)...)
	diagnostics = append(diagnostics, htmlTypeDiagnostics(uri, text)...)
	for _, issue := range compiler.UnsupportedClasses([][]byte{[]byte(text)}, nil) {
		start := byteOffsetToPosition(text, issue.Offset)
		diagnostics = append(diagnostics, diagnostic{
			Range:    protocolRange{Start: start, End: byteOffsetToPosition(text, issue.Offset+len(issue.Name))},
			Severity: 2, Source: "northframe",
			Message: fmt.Sprintf("unsupported utility class %q; define it in <style> or use a supported Tailwind utility", issue.Name),
		})
	}
	path := documentPath(uri)
	base := filepath.Base(path)
	if base == "layout.north" && !strings.Contains(text, "<slot />") && !strings.Contains(text, "<slot/>") {
		diagnostics = append(diagnostics, diagnostic{
			Range:    protocolRange{Start: position{}, End: position{Character: 1}},
			Severity: 1, Source: "northframe", Message: "layout.north must contain <slot />",
		})
	}
	if base == "page.north" || base == "layout.north" {
		sidecar := path + ".go"
		if _, err := os.Stat(sidecar); os.IsNotExist(err) {
			diagnostics = append(diagnostics, diagnostic{
				Range:    protocolRange{Start: position{}, End: position{Character: 1}},
				Severity: 2, Source: "northframe",
				Message: fmt.Sprintf("missing typed sidecar %s", filepath.Base(sidecar)),
			})
		}
	}
	return diagnostics
}

func htmlTypeDiagnostics(uri, text string) []diagnostic {
	var diagnostics []diagnostic
	for _, match := range serverHTMLExpression.FindAllStringSubmatchIndex(text, -1) {
		expressionStart := match[2]
		expressionEnd := match[3]
		value, ok := templateValueAt(uri, text, expressionStart)
		if !ok || value.Type == "web.SafeHTML" {
			continue
		}
		diagnostics = append(diagnostics, diagnostic{
			Range: protocolRange{
				Start: byteOffsetToPosition(text, expressionStart),
				End:   byteOffsetToPosition(text, expressionEnd),
			},
			Severity: 1,
			Source:   "northframe",
			Message:  fmt.Sprintf("{html ...} requires web.SafeHTML, but %s has type %s; sanitize the value and wrap it with web.SafeHTMLFromSanitized", value.Name, value.Type),
		})
	}
	return diagnostics
}

func propsTypeDiagnostics(uri, text string) []diagnostic {
	block, blockStart, _, found := propsBlockContent(text)
	if !found {
		return nil
	}
	if match := frontmatterPropsInterface.FindStringSubmatchIndex(block); len(match) >= 4 {
		blockStart += match[2]
		block = block[match[2]:match[3]]
	}
	imports := map[string]propsImport{}
	for _, imported := range propsImports(text) {
		imports[imported.Alias] = imported
	}
	project := map[string][]projectImport{}
	for _, imported := range projectImports(uri) {
		project[imported.Alias] = append(project[imported.Alias], imported)
	}
	seen := map[string]bool{}
	var diagnostics []diagnostic
	for _, match := range qualifiedGoType.FindAllStringSubmatchIndex(block, -1) {
		alias := block[match[2]:match[3]]
		name := block[match[4]:match[5]]
		key := alias + "." + name
		if seen[key] {
			continue
		}
		seen[key] = true
		startOffset := blockStart + match[2]
		endOffset := blockStart + match[5]
		if imported, ok := imports[alias]; ok {
			known := false
			for _, symbol := range importedGoTypes(uri, imported) {
				if symbol.Name == name {
					known = true
					break
				}
			}
			if !known && strings.HasPrefix(imported.Path, modulePathFor(uri)+"/") {
				diagnostics = append(diagnostics, diagnostic{
					Range:    protocolRange{Start: byteOffsetToPosition(text, startOffset), End: byteOffsetToPosition(text, endOffset)},
					Severity: 1, Source: "northframe", Message: fmt.Sprintf("%s does not export Go type %s", imported.Path, name),
				})
			}
			continue
		}
		message := fmt.Sprintf("Go package %s is not imported", alias)
		if candidates := project[alias]; len(candidates) == 1 {
			message += fmt.Sprintf("; choose a %s type completion to add `import %s %q` automatically", alias, alias, candidates[0].Path)
		}
		diagnostics = append(diagnostics, diagnostic{
			Range:    protocolRange{Start: byteOffsetToPosition(text, startOffset), End: byteOffsetToPosition(text, blockStart+match[3])},
			Severity: 1, Source: "northframe", Message: message,
		})
	}
	return diagnostics
}

func modulePathFor(uri string) string {
	_, module := goModuleFor(documentPath(uri))
	return strings.TrimRight(module, "/")
}

func validationErrorOffset(message string) int {
	match := bytePosition.FindStringSubmatch(message)
	if len(match) != 2 {
		return 0
	}
	offset, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}
	return offset
}
