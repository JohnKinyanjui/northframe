package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type northDocument struct {
	URI    string
	Source string
}

func (current *server) references(id json.RawMessage, raw json.RawMessage) error {
	var params textDocumentPosition
	if err := json.Unmarshal(raw, &params); err != nil {
		return err
	}
	text := current.document(params.TextDocument.URI)
	offset := positionToByteOffset(text, params.Position)
	return current.reply(id, referenceLocations(params.TextDocument.URI, text, offset))
}

func referenceLocations(uri, source string, offset int) []map[string]any {
	if target, ok := componentPropAt(uri, source, offset); ok {
		return componentPropReferences(uri, source, target)
	}
	word := wordAt(source, offset)
	if component, ok := findComponent(uri, tokenRoot(word)); ok {
		return componentReferences(uri, source, component)
	}
	name := tokenRoot(strings.TrimPrefix(word, "Props."))
	if field, ok := findDocumentProp(uri, source, name); ok {
		locations := []map[string]any{{"uri": field.URI, "range": field.Range}}
		pattern := regexp.MustCompile(`\bProps\.` + regexp.QuoteMeta(field.Name) + `\b`)
		for _, match := range pattern.FindAllStringIndex(source, -1) {
			start := match[1] - len(field.Name)
			locations = appendUniqueLocation(locations, uri, protocolRange{
				Start: byteOffsetToPosition(source, start), End: byteOffsetToPosition(source, match[1]),
			})
		}
		return locations
	}
	if target, ok := refactorTarget(uri, source, offset); ok {
		locations := make([]map[string]any, 0, len(target.Occurrences))
		for _, occurrence := range target.Occurrences {
			locations = appendUniqueLocation(locations, uri, occurrence)
		}
		return locations
	}
	return nil
}

func componentReferences(uri, currentSource string, component projectComponent) []map[string]any {
	pattern := regexp.MustCompile(`<` + regexp.QuoteMeta(component.Name) + `\b`)
	var locations []map[string]any
	for _, document := range projectNorthDocuments(uri, currentSource) {
		for _, match := range pattern.FindAllStringIndex(document.Source, -1) {
			start := match[0] + 1
			locations = appendUniqueLocation(locations, document.URI, protocolRange{
				Start: byteOffsetToPosition(document.Source, start), End: byteOffsetToPosition(document.Source, match[1]),
			})
		}
	}
	return locations
}

func componentPropReferences(uri, currentSource string, target componentPropTarget) []map[string]any {
	locations := []map[string]any{{"uri": target.Field.URI, "range": target.Field.Range}}
	tagPattern := regexp.MustCompile(`(?s)<` + regexp.QuoteMeta(target.Component.Name) + `\b[^>]*>`)
	attributePattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(target.Field.Name) + `\s*=`)
	for _, document := range projectNorthDocuments(uri, currentSource) {
		for _, tag := range tagPattern.FindAllStringIndex(document.Source, -1) {
			for _, attribute := range attributePattern.FindAllStringIndex(document.Source[tag[0]:tag[1]], -1) {
				start := tag[0] + attribute[0]
				locations = appendUniqueLocation(locations, document.URI, protocolRange{
					Start: byteOffsetToPosition(document.Source, start), End: byteOffsetToPosition(document.Source, start+len(target.Field.Name)),
				})
			}
		}
	}
	return locations
}

func projectNorthDocuments(uri, currentSource string) []northDocument {
	root, _ := goModuleFor(documentPath(uri))
	if root == "" {
		return []northDocument{{URI: uri, Source: currentSource}}
	}
	webRoot := filepath.Join(root, "web")
	var documents []northDocument
	_ = filepath.WalkDir(webRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Ext(path) != ".north" {
			return nil
		}
		documentURI := documentURI(path)
		if documentURI == uri {
			documents = append(documents, northDocument{URI: uri, Source: currentSource})
			return nil
		}
		contents, readErr := os.ReadFile(path)
		if readErr == nil {
			documents = append(documents, northDocument{URI: documentURI, Source: string(contents)})
		}
		return nil
	})
	return documents
}

func appendUniqueLocation(items []map[string]any, uri string, target protocolRange) []map[string]any {
	for _, item := range items {
		if item["uri"] == uri && item["range"] == target {
			return items
		}
	}
	return append(items, map[string]any{"uri": uri, "range": target})
}
