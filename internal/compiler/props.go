package compiler

import (
	"fmt"
	pathpkg "path"
	"strconv"
	"strings"
)

func parseProps(block string) ([]prop, []componentImport, error) {
	var props []prop
	var imports []componentImport
	for lineNumber, raw := range strings.Split(block, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "import ") {
			parsed, err := parseComponentImport(strings.TrimSpace(strings.TrimPrefix(line, "import")))
			if err != nil {
				return nil, nil, fmt.Errorf("invalid import on line %d: %w", lineNumber+1, err)
			}
			imports = append(imports, parsed)
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 2 || !identifier.MatchString(parts[0]) {
			return nil, nil, fmt.Errorf("invalid prop declaration on line %d: use `Name type`", lineNumber+1)
		}
		props = append(props, prop{Name: parts[0], Type: parts[1]})
	}
	return props, imports, nil
}

func parseComponentImport(value string) (componentImport, error) {
	parts := strings.Fields(value)
	if len(parts) < 1 || len(parts) > 2 {
		return componentImport{}, fmt.Errorf("use `import \"module/path\"` or `import alias \"module/path\"`")
	}
	quotedPath := parts[len(parts)-1]
	importPath, err := strconv.Unquote(quotedPath)
	if err != nil || importPath == "" {
		return componentImport{}, fmt.Errorf("import path must be quoted")
	}
	alias := pathpkg.Base(importPath)
	if len(parts) == 2 {
		alias = parts[0]
	}
	if !identifier.MatchString(alias) || alias == "web" {
		return componentImport{}, fmt.Errorf("invalid or reserved import alias %q", alias)
	}
	return componentImport{Alias: alias, Path: importPath}, nil
}
