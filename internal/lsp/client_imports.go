package lsp

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/JohnKinyanjui/northframe/internal/dependencies"
)

var (
	lspTypeScriptBlock = regexp.MustCompile(`(?is)<script\b[^>]*\blang\s*=\s*["']ts["'][^>]*>(.*?)</script\s*>`)
	clientImportPrefix = regexp.MustCompile(`(?:\bfrom\s+|\bimport\s*)["']([^"']*)$`)
)

func clientImportCompletionItems(uri, text string, offset int) []map[string]any {
	if !insideTypeScript(text, offset) {
		return nil
	}
	lineStart := strings.LastIndex(text[:offset], "\n") + 1
	line := text[lineStart:offset]
	match := clientImportPrefix.FindStringSubmatchIndex(line)
	if len(match) < 4 {
		return nil
	}
	prefix := line[match[2]:match[3]]
	replaceStart := lineStart + match[2]
	projectRoot, err := dependencies.FindProjectRoot(documentPath(uri))
	if err != nil {
		return nil
	}
	manifest, err := dependencies.LoadManifest(projectRoot)
	if err != nil {
		return nil
	}
	candidates := dependencies.DependencyNames(manifest)
	candidates = append(candidates, localClientImports(projectRoot, manifest.Client.Source)...)
	sort.Strings(candidates)
	items := make([]map[string]any, 0, len(candidates))
	for _, candidate := range candidates {
		if prefix != "" && !strings.HasPrefix(candidate, prefix) {
			continue
		}
		items = append(items, map[string]any{
			"label": candidate, "kind": 9, "detail": "Northframe client import",
			"textEdit": map[string]any{
				"range":   protocolRange{Start: byteOffsetToPosition(text, replaceStart), End: byteOffsetToPosition(text, offset)},
				"newText": candidate,
			},
		})
	}
	return items
}

func insideTypeScript(text string, offset int) bool {
	for _, match := range lspTypeScriptBlock.FindAllStringSubmatchIndex(text, -1) {
		if len(match) >= 4 && offset >= match[2] && offset <= match[3] {
			return true
		}
	}
	return false
}

func localClientImports(root, configured string) []string {
	clientRoot := filepath.Join(root, filepath.FromSlash(configured))
	seen := map[string]bool{}
	_ = filepath.WalkDir(clientRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		extension := strings.ToLower(filepath.Ext(path))
		if extension != ".ts" && extension != ".tsx" && extension != ".js" && extension != ".jsx" {
			return nil
		}
		relative, relErr := filepath.Rel(clientRoot, path)
		if relErr != nil {
			return nil
		}
		relative = strings.TrimSuffix(filepath.ToSlash(relative), extension)
		relative = strings.TrimSuffix(relative, "/index")
		seen["$client/"+relative] = true
		return nil
	})
	result := make([]string, 0, len(seen))
	for name := range seen {
		result = append(result, name)
	}
	return result
}
