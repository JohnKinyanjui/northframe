package lsp

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var inferredRouteProp = regexp.MustCompile(`\{\s*Props\.([A-Z][A-Za-z0-9_]*)\s*\}`)

type propField struct {
	Name  string
	Type  string
	URI   string
	Range protocolRange
}

func sidecarProps(templateURI string) []propField {
	path := documentPath(templateURI)
	if contents, err := os.ReadFile(path); err == nil {
		if fields := templateProps(templateURI, string(contents)); len(fields) > 0 {
			return fields
		}
	}
	base := filepath.Base(path)
	propsName := "PageProps"
	if base == "layout.north" {
		propsName = "LayoutProps"
	} else if base != "page.north" {
		return nil
	}
	sidecarPath := path + ".go"
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, sidecarPath, nil, 0)
	if err != nil {
		return nil
	}
	var fields []propField
	ast.Inspect(file, func(node ast.Node) bool {
		typeSpec, ok := node.(*ast.TypeSpec)
		if !ok || typeSpec.Name.Name != propsName {
			return true
		}
		structure, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return false
		}
		for _, field := range structure.Fields.List {
			var typeSource bytes.Buffer
			_ = format.Node(&typeSource, fileSet, field.Type)
			for _, name := range field.Names {
				start := fileSet.Position(name.Pos())
				end := fileSet.Position(name.End())
				fields = append(fields, propField{
					Name: name.Name, Type: typeSource.String(), URI: documentURI(sidecarPath),
					Range: protocolRange{
						Start: position{Line: start.Line - 1, Character: start.Column - 1},
						End:   position{Line: end.Line - 1, Character: end.Column - 1},
					},
				})
			}
		}
		return false
	})
	return fields
}

func documentProps(uri, source string) []propField {
	if fields := templateProps(uri, source); len(fields) > 0 {
		return fields
	}
	return sidecarProps(uri)
}

func templateProps(uri, source string) []propField {
	if block, blockStart, _, found := propsBlockContent(source); found {
		var fields []propField
		offset := 0
		for _, raw := range strings.SplitAfter(block, "\n") {
			line := strings.TrimSuffix(raw, "\n")
			trimmed := strings.TrimSpace(line)
			parts := strings.Fields(trimmed)
			if len(parts) == 2 && !strings.HasPrefix(trimmed, "import ") && !strings.HasPrefix(trimmed, "//") {
				nameOffset := blockStart + offset + strings.Index(line, parts[0])
				fields = append(fields, propField{
					Name: parts[0], Type: parts[1], URI: uri,
					Range: protocolRange{Start: byteOffsetToPosition(source, nameOffset), End: byteOffsetToPosition(source, nameOffset+len(parts[0]))},
				})
			}
			offset += len(raw)
		}
		return fields
	}
	seen := map[string]bool{}
	var fields []propField
	for _, match := range inferredRouteProp.FindAllStringSubmatchIndex(source, -1) {
		name := source[match[2]:match[3]]
		if seen[name] {
			continue
		}
		seen[name] = true
		fields = append(fields, propField{
			Name: name, Type: "string", URI: uri,
			Range: protocolRange{Start: byteOffsetToPosition(source, match[2]), End: byteOffsetToPosition(source, match[3])},
		})
	}
	return fields
}

func findProp(uri, name string) (propField, bool) {
	for _, field := range sidecarProps(uri) {
		if field.Name == name {
			return field, true
		}
	}
	return propField{}, false
}

func findDocumentProp(uri, source, name string) (propField, bool) {
	for _, field := range documentProps(uri, source) {
		if field.Name == name {
			return field, true
		}
	}
	return propField{}, false
}
