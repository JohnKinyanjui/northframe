package lsp

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

var (
	frontmatterPropsBlock     = regexp.MustCompile(`(?ms)\A[ \t]*---[ \t]*\r?\n(.*?)^[ \t]*---[ \t]*(?:\r?\n|\z)`)
	frontmatterPropsInterface = regexp.MustCompile(`(?ms)^[ \t]*interface[ \t]+Props[ \t]*\{[ \t]*\r?\n(.*?)^[ \t]*\}[ \t]*(?:\r?\n|\z)`)
	legacyScriptPropsBlock    = regexp.MustCompile(`(?s)<script\s+context=["']props["']\s*>(.*?)</script>`)
	legacyInterfacePropsBlock = regexp.MustCompile(`(?ms)^[ \t]*interface[ \t]+props[ \t]*\{[ \t]*\r?\n(.*?)^[ \t]*\}[ \t]*(?:\r?\n|$)`)
	componentDispatch         = regexp.MustCompile(`\bdispatch\s*\(\s*["']([A-Za-z][A-Za-z0-9_-]*)["']`)
)

type projectComponent struct {
	Name    string
	Path    string
	Props   []string
	Events  []string
	HasSlot bool
}

func projectComponents(uri string) []projectComponent {
	root := componentsRoot(documentPath(uri))
	if root == "" {
		return nil
	}
	var result []projectComponent
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Ext(path) != ".north" {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		contents, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		result = append(result, projectComponent{
			Name: componentTypeName(strings.TrimSuffix(filepath.ToSlash(relative), ".north")),
			Path: path, Props: componentPropNames(string(contents)), Events: componentEventNames(string(contents)), HasSlot: strings.Contains(string(contents), "<slot"),
		})
		return nil
	})
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func componentsRoot(path string) string {
	if path == "" {
		return ""
	}
	current := filepath.Dir(path)
	for {
		if filepath.Base(current) == "routes" || filepath.Base(current) == "components" {
			candidate := filepath.Join(filepath.Dir(current), "components")
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				return candidate
			}
		}
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			candidate := filepath.Join(current, "components")
			if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
				return candidate
			}
			return ""
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func componentPropNames(source string) []string {
	block, _, _, found := propsBlockContent(source)
	if !found {
		return nil
	}
	if match := frontmatterPropsInterface.FindStringSubmatch(block); len(match) == 2 {
		block = match[1]
	}
	var result []string
	for _, raw := range strings.Split(block, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "import ") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			result = append(result, parts[0])
		}
	}
	return result
}

func componentEventNames(source string) []string {
	seen := map[string]bool{}
	var result []string
	for _, match := range componentDispatch.FindAllStringSubmatch(source, -1) {
		if !seen[match[1]] {
			seen[match[1]] = true
			result = append(result, match[1])
		}
	}
	sort.Strings(result)
	return result
}

func propsBlockContent(source string) (content string, start, end int, found bool) {
	for _, pattern := range []*regexp.Regexp{frontmatterPropsBlock, legacyScriptPropsBlock, legacyInterfacePropsBlock} {
		match := pattern.FindStringSubmatchIndex(source)
		if len(match) >= 4 {
			return source[match[2]:match[3]], match[2], match[3], true
		}
	}
	return "", 0, 0, false
}

func componentTypeName(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	var output strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		output.WriteRune(unicode.ToUpper(runes[0]))
		output.WriteString(string(runes[1:]))
	}
	return output.String()
}

func findComponent(uri, name string) (projectComponent, bool) {
	for _, current := range projectComponents(uri) {
		if current.Name == name {
			return current, true
		}
	}
	return projectComponent{}, false
}
