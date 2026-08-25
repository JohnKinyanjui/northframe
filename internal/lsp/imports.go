package lsp

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type projectImport struct {
	Alias string
	Path  string
}

func projectImports(uri string) []projectImport {
	root, modulePath := goModuleFor(documentPath(uri))
	if root == "" || modulePath == "" {
		return []projectImport{{Alias: "time", Path: "time"}}
	}
	currentDirectory := filepath.Dir(documentPath(uri))
	seenDirectories := map[string]bool{}
	seenPaths := map[string]bool{}
	var imports []projectImport
	for _, current := range resolvedGoPackages(uri) {
		if current.Path == modulePath || strings.HasPrefix(current.Path, strings.TrimRight(modulePath, "/")+"/") {
			continue
		}
		imports = append(imports, projectImport{Alias: current.Alias, Path: current.Path})
		seenPaths[current.Path] = true
	}
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if path != root && (entry.Name() == ".git" || entry.Name() == ".generated" || entry.Name() == "vendor" || entry.Name() == "node_modules") {
				return filepath.SkipDir
			}
			if path != root {
				if _, nestedModuleErr := os.Stat(filepath.Join(path, "go.mod")); nestedModuleErr == nil {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		directory := filepath.Dir(path)
		if seenDirectories[directory] || directory == currentDirectory {
			return nil
		}
		seenDirectories[directory] = true
		file, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly)
		if parseErr != nil || file.Name.Name == "main" {
			return nil
		}
		relative, relErr := filepath.Rel(root, directory)
		if relErr != nil || relative == "." || strings.HasPrefix(filepath.ToSlash(relative), ".generated/") {
			return nil
		}
		importPath := strings.TrimRight(modulePath, "/") + "/" + filepath.ToSlash(relative)
		if !seenPaths[importPath] {
			imports = append(imports, projectImport{Alias: file.Name.Name, Path: importPath})
			seenPaths[importPath] = true
		}
		return nil
	})
	sort.Slice(imports, func(i, j int) bool { return imports[i].Path < imports[j].Path })
	return imports
}

func goModuleFor(path string) (string, string) {
	current := filepath.Dir(path)
	for {
		contents, err := os.ReadFile(filepath.Join(current, "go.mod"))
		if err == nil {
			for _, raw := range strings.Split(string(contents), "\n") {
				line := strings.TrimSpace(raw)
				if strings.HasPrefix(line, "module ") {
					return current, strings.TrimSpace(strings.TrimPrefix(line, "module "))
				}
			}
			return "", ""
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", ""
		}
		current = parent
	}
}

func propsImportCompletionItems(uri, text string, offset int) []map[string]any {
	if !insidePropsBlock(text, offset) {
		return nil
	}
	lineStart := strings.LastIndex(text[:offset], "\n") + 1
	line := text[lineStart:offset]
	trimmed := strings.TrimSpace(line)
	if trimmed != "" && !strings.HasPrefix(trimmed, "import") {
		return nil
	}
	replaceStart := offset
	wholeLine := true
	if index := strings.Index(line, "import"); index >= 0 {
		replaceStart = lineStart + index + len("import")
		for replaceStart < offset && (text[replaceStart] == ' ' || text[replaceStart] == '\t') {
			replaceStart++
		}
		wholeLine = false
	}
	start := byteOffsetToPosition(text, replaceStart)
	end := byteOffsetToPosition(text, offset)
	items := make([]map[string]any, 0)
	for _, imported := range projectImports(uri) {
		newText := imported.Alias + " \"" + imported.Path + "\""
		if wholeLine {
			newText = "import " + newText
		}
		items = append(items, map[string]any{
			"label": imported.Path, "kind": 9,
			"detail": "Import Go type package as " + imported.Alias,
			"textEdit": map[string]any{
				"range": protocolRange{Start: start, End: end}, "newText": newText,
			},
		})
	}
	return items
}

func insidePropsBlock(text string, offset int) bool {
	_, start, end, found := propsBlockContent(text)
	return found && offset >= start && offset <= end
}
