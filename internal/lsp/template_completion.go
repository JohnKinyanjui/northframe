package lsp

import (
	"regexp"
	"strings"
)

var templateExpressionPrefix = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_.]*)$`)

func templateFieldCompletionItems(uri, source string, offset int) []map[string]any {
	if offset < 0 || offset > len(source) || insidePropsBlock(source, offset) {
		return nil
	}
	lineStart := strings.LastIndex(source[:offset], "\n") + 1
	match := templateExpressionPrefix.FindStringSubmatchIndex(source[lineStart:offset])
	if match == nil {
		return nil
	}
	prefix := source[lineStart+match[2] : lineStart+match[3]]
	dot := strings.LastIndex(prefix, ".")
	if dot < 0 {
		return nil
	}
	receiver := prefix[:dot]
	partial := prefix[dot+1:]
	replaceStart := offset - len(partial)
	start := byteOffsetToPosition(source, replaceStart)
	end := byteOffsetToPosition(source, offset)
	var fields []goFieldSymbol
	if receiver == "Props" {
		for _, field := range documentProps(uri, source) {
			fields = append(fields, goFieldSymbol{Name: field.Name, Type: field.Type, URI: field.URI, Range: field.Range})
		}
	} else {
		fields = templateFieldsFor(uri, source, receiver, offset)
	}
	items := make([]map[string]any, 0, len(fields))
	for _, field := range fields {
		item := map[string]any{
			"label": field.Name, "kind": 5, "detail": field.Type + " — Go field",
			"textEdit": map[string]any{"range": protocolRange{Start: start, End: end}, "newText": field.Name},
		}
		if field.Documentation != "" {
			item["documentation"] = field.Documentation
		}
		items = append(items, item)
	}
	return items
}
