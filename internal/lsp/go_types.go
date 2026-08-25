package lsp

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	pathpkg "path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	propsImportLine   = regexp.MustCompile(`(?m)^[ \t]*import[ \t]+(?:([A-Za-z_][A-Za-z0-9_]*)[ \t]+)?("[^"]+")[ \t]*$`)
	qualifiedGoType   = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\.([A-Za-z_][A-Za-z0-9_]*)\b`)
	qualifiedGoPrefix = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)\.([A-Za-z_][A-Za-z0-9_]*)?$`)
)

type propsImport struct {
	Alias string
	Path  string
}

type goTypeSymbol struct {
	Name          string
	Kind          int
	Declaration   string
	Documentation string
	URI           string
	Range         protocolRange
	Fields        []goFieldSymbol
}

type goFieldSymbol struct {
	Name          string
	Type          string
	Documentation string
	URI           string
	Range         protocolRange
}

func propsImports(text string) []propsImport {
	content, _, _, found := propsBlockContent(text)
	if !found {
		return nil
	}
	imports := make([]propsImport, 0)
	for _, match := range propsImportLine.FindAllStringSubmatch(content, -1) {
		importPath, err := strconv.Unquote(match[2])
		if err != nil || importPath == "" {
			continue
		}
		alias := match[1]
		if alias == "" {
			alias = pathpkg.Base(importPath)
		}
		imports = append(imports, propsImport{Alias: alias, Path: importPath})
	}
	return imports
}

func importedGoTypes(uri string, imported propsImport) []goTypeSymbol {
	root, modulePath := goModuleFor(documentPath(uri))
	if root == "" || modulePath == "" {
		return nil
	}
	directory := ""
	if imported.Path == modulePath {
		directory = root
	} else if strings.HasPrefix(imported.Path, strings.TrimRight(modulePath, "/")+"/") {
		relative := strings.TrimPrefix(imported.Path, strings.TrimRight(modulePath, "/")+"/")
		directory = filepath.Join(root, filepath.FromSlash(relative))
	} else {
		directory = goPackageDirectory(uri, imported.Path)
	}
	if directory == "" {
		return nil
	}

	fileSet := token.NewFileSet()
	packages, err := parser.ParseDir(fileSet, directory, func(info fs.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return nil
	}
	packageNames := make([]string, 0, len(packages))
	for name := range packages {
		packageNames = append(packageNames, name)
	}
	sort.Strings(packageNames)
	var symbols []goTypeSymbol
	for _, packageName := range packageNames {
		parsedPackage := packages[packageName]
		fileNames := make([]string, 0, len(parsedPackage.Files))
		for fileName := range parsedPackage.Files {
			fileNames = append(fileNames, fileName)
		}
		sort.Strings(fileNames)
		for _, fileName := range fileNames {
			file := parsedPackage.Files[fileName]
			for _, declaration := range file.Decls {
				general, ok := declaration.(*ast.GenDecl)
				if !ok || general.Tok != token.TYPE {
					continue
				}
				for _, specification := range general.Specs {
					typeSpec, ok := specification.(*ast.TypeSpec)
					if !ok || !typeSpec.Name.IsExported() {
						continue
					}
					var formatted bytes.Buffer
					_ = format.Node(&formatted, fileSet, typeSpec)
					positionStart := fileSet.Position(typeSpec.Name.Pos())
					positionEnd := fileSet.Position(typeSpec.Name.End())
					documentation := ""
					if typeSpec.Doc != nil {
						documentation = strings.TrimSpace(typeSpec.Doc.Text())
					} else if general.Doc != nil {
						documentation = strings.TrimSpace(general.Doc.Text())
					}
					kind := 7
					switch typeSpec.Type.(type) {
					case *ast.InterfaceType:
						kind = 8
					case *ast.StructType:
						kind = 22
					}
					symbol := goTypeSymbol{
						Name: typeSpec.Name.Name, Kind: kind,
						Declaration: "type " + formatted.String(), Documentation: documentation,
						URI: documentURI(fileName),
						Range: protocolRange{
							Start: position{Line: positionStart.Line - 1, Character: positionStart.Column - 1},
							End:   position{Line: positionEnd.Line - 1, Character: positionEnd.Column - 1},
						},
					}
					if structure, ok := typeSpec.Type.(*ast.StructType); ok {
						for _, field := range structure.Fields.List {
							var fieldType bytes.Buffer
							_ = format.Node(&fieldType, fileSet, field.Type)
							documentation := ""
							if field.Doc != nil {
								documentation = strings.TrimSpace(field.Doc.Text())
							} else if field.Comment != nil {
								documentation = strings.TrimSpace(field.Comment.Text())
							}
							for _, name := range field.Names {
								if !name.IsExported() {
									continue
								}
								start := fileSet.Position(name.Pos())
								end := fileSet.Position(name.End())
								symbol.Fields = append(symbol.Fields, goFieldSymbol{
									Name: name.Name, Type: fieldType.String(), Documentation: documentation,
									URI: documentURI(fileName),
									Range: protocolRange{
										Start: position{Line: start.Line - 1, Character: start.Column - 1},
										End:   position{Line: end.Line - 1, Character: end.Column - 1},
									},
								})
							}
						}
					}
					symbols = append(symbols, symbol)
				}
			}
		}
	}
	sort.Slice(symbols, func(i, j int) bool { return symbols[i].Name < symbols[j].Name })
	return symbols
}

func goTypeCompletionItems(uri, text string, offset int) []map[string]any {
	if !insidePropsBlock(text, offset) {
		return nil
	}
	lineStart := strings.LastIndex(text[:offset], "\n") + 1
	match := qualifiedGoPrefix.FindStringSubmatchIndex(text[lineStart:offset])
	if match == nil {
		return unqualifiedGoTypeCompletionItems(uri, text, offset)
	}
	alias := text[lineStart:offset][match[2]:match[3]]
	partialStart := offset
	if match[4] >= 0 {
		partialStart = lineStart + match[4]
	}
	imports := propsImports(text)
	var candidates []propsImport
	for _, imported := range imports {
		if imported.Alias == alias {
			candidates = append(candidates, imported)
		}
	}
	if len(candidates) == 0 {
		candidates = missingProjectImports(uri, imports, alias)
	}
	start := byteOffsetToPosition(text, partialStart)
	end := byteOffsetToPosition(text, offset)
	partial := strings.ToLower(text[partialStart:offset])
	items := make([]map[string]any, 0)
	for _, imported := range candidates {
		for _, symbol := range importedGoTypes(uri, imported) {
			if partial != "" && !strings.HasPrefix(strings.ToLower(symbol.Name), partial) {
				continue
			}
			item := map[string]any{
				"label": symbol.Name, "kind": symbol.Kind,
				"detail":   symbol.Declaration + " — " + imported.Path,
				"textEdit": map[string]any{"range": protocolRange{Start: start, End: end}, "newText": symbol.Name},
			}
			if edit, ok := missingImportEdit(text, imported); ok {
				item["additionalTextEdits"] = []map[string]any{edit}
				item["detail"] = symbol.Declaration + " — auto-import " + imported.Path
			}
			items = append(items, item)
		}
	}
	if len(items) > 0 {
		return items
	}
	return unqualifiedGoTypeCompletionItems(uri, text, offset)
}

func missingProjectImports(uri string, existing []propsImport, alias string) []propsImport {
	known := map[string]bool{}
	for _, imported := range existing {
		known[imported.Path] = true
	}
	var result []propsImport
	for _, imported := range projectImports(uri) {
		if imported.Alias == alias && !known[imported.Path] {
			result = append(result, propsImport{Alias: imported.Alias, Path: imported.Path})
		}
	}
	return result
}

func unqualifiedGoTypeCompletionItems(uri, text string, offset int) []map[string]any {
	lineStart := strings.LastIndex(text[:offset], "\n") + 1
	line := text[lineStart:offset]
	match := regexp.MustCompile(`([A-Z][A-Za-z0-9_]*)$`).FindStringSubmatchIndex(line)
	trimmed := strings.TrimSpace(line)
	if match == nil || len(strings.Fields(trimmed)) < 2 || strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "interface ") {
		return nil
	}
	partialStart := lineStart + match[2]
	partial := strings.ToLower(text[partialStart:offset])
	start := byteOffsetToPosition(text, partialStart)
	end := byteOffsetToPosition(text, offset)
	existing := propsImports(text)
	known := map[string]bool{}
	for _, imported := range existing {
		known[imported.Path] = true
	}
	var items []map[string]any
	_, modulePath := goModuleFor(documentPath(uri))
	modulePrefix := strings.TrimRight(modulePath, "/") + "/"
	for _, projectImport := range projectImports(uri) {
		if projectImport.Path != modulePath && !strings.HasPrefix(projectImport.Path, modulePrefix) {
			continue
		}
		imported := propsImport{Alias: projectImport.Alias, Path: projectImport.Path}
		for _, symbol := range importedGoTypes(uri, imported) {
			if !strings.HasPrefix(strings.ToLower(symbol.Name), partial) {
				continue
			}
			item := map[string]any{
				"label": symbol.Name, "kind": symbol.Kind,
				"filterText": symbol.Name + " " + imported.Alias + "." + symbol.Name,
				"detail":     symbol.Declaration + " — " + imported.Path,
				"textEdit":   map[string]any{"range": protocolRange{Start: start, End: end}, "newText": imported.Alias + "." + symbol.Name},
			}
			if !known[imported.Path] {
				if edit, ok := missingImportEdit(text, imported); ok {
					item["additionalTextEdits"] = []map[string]any{edit}
					item["detail"] = symbol.Declaration + " — auto-import " + imported.Path
				}
			}
			items = append(items, item)
		}
	}
	return items
}

func missingImportEdit(text string, imported propsImport) (map[string]any, bool) {
	for _, existing := range propsImports(text) {
		if existing.Path == imported.Path || existing.Alias == imported.Alias {
			return nil, false
		}
	}
	block, start, _, found := propsBlockContent(text)
	if !found {
		return nil, false
	}
	line := `import ` + imported.Alias + ` "` + imported.Path + `"`
	matches := propsImportLine.FindAllStringIndex(block, -1)
	insertion := start
	newText := line + "\n\n"
	if len(matches) > 0 {
		last := matches[len(matches)-1]
		insertion = start + last[1]
		newText = "\n" + line
	} else if interfaceOffset := strings.Index(block, "interface Props"); interfaceOffset >= 0 {
		lineStart := strings.LastIndex(block[:interfaceOffset], "\n") + 1
		insertion = start + lineStart
	}
	where := byteOffsetToPosition(text, insertion)
	return map[string]any{"range": protocolRange{Start: where, End: where}, "newText": newText}, true
}

func importedGoTypeAt(uri, text string, offset int) (goTypeSymbol, bool) {
	lineStart := strings.LastIndex(text[:offset], "\n") + 1
	lineEndOffset := strings.Index(text[offset:], "\n")
	lineEnd := len(text)
	if lineEndOffset >= 0 {
		lineEnd = offset + lineEndOffset
	}
	line := text[lineStart:lineEnd]
	lineOffset := offset - lineStart
	for _, match := range qualifiedGoType.FindAllStringSubmatchIndex(line, -1) {
		if lineOffset < match[0] || lineOffset > match[1] {
			continue
		}
		alias := line[match[2]:match[3]]
		name := line[match[4]:match[5]]
		for _, imported := range propsImports(text) {
			if imported.Alias != alias {
				continue
			}
			for _, symbol := range importedGoTypes(uri, imported) {
				if symbol.Name == name {
					return symbol, true
				}
			}
		}
	}
	return goTypeSymbol{}, false
}
