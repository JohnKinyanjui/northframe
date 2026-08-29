package compiler

import (
	"bytes"
	"fmt"
	"go/format"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const routePropsFilename = "props_generated.go"

var (
	serverPropsValue = regexp.MustCompile(`(?:\$\{|\{)Props\.([A-Z][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*)\}`)
	plainServerValue = regexp.MustCompile(`(?:\$\{|\{)([A-Z][A-Za-z0-9_]*)\}`)
	serverIf         = regexp.MustCompile(`\{if\s+Props\.([A-Z][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*)\s*\}`)
	serverFor        = regexp.MustCompile(`\{for\s+[a-z_][A-Za-z0-9_]*\s*:=\s*range\s+Props\.([^}]+)\}`)
	clientPropValue  = regexp.MustCompile(`\bprops\.([A-Z][A-Za-z0-9_]*)(\.[A-Za-z_][A-Za-z0-9_.]*)?`)
)

func resolveRouteProps(path string, source []byte, sidecar sidecarInfo) ([]prop, []componentImport, bool, error) {
	block, _, hasContract, err := extractPropsBlock(string(source))
	if err != nil {
		return nil, nil, false, fmt.Errorf("%s: %w", path, err)
	}
	if sidecar.DeclaresProps {
		if hasContract {
			return nil, nil, false, fmt.Errorf("%s declares props in both the template and sidecar; remove type %s from %s.north.go", path, sidecarPropsName(path), strings.TrimSuffix(filepath.Base(path), ".north"))
		}
		return sidecar.Props, sidecar.Imports, false, nil
	}
	if hasContract {
		props, imports, err := parseProps(block)
		if err != nil {
			return nil, nil, false, fmt.Errorf("%s props: %w", path, err)
		}
		return props, imports, true, nil
	}
	props, err := inferRouteProps(path, string(source))
	if err != nil {
		return nil, nil, false, err
	}
	return props, nil, true, nil
}

func sidecarPropsName(path string) string {
	if filepath.Base(path) == "layout.north" {
		return "LayoutProps"
	}
	return "PageProps"
}

func inferRouteProps(path, source string) ([]prop, error) {
	types := map[string]string{}
	add := func(name, kind, expression string) error {
		if strings.Contains(expression, ".") {
			return fmt.Errorf("%s cannot infer the Go type of %s; add an `interface Props { ... }` frontmatter contract", path, expression)
		}
		if existing := types[name]; existing != "" && existing != kind {
			return fmt.Errorf("%s uses %s as both %s and %s; declare its type in a props contract", path, name, existing, kind)
		}
		types[name] = kind
		return nil
	}
	for _, match := range serverPropsValue.FindAllStringSubmatch(source, -1) {
		root := strings.Split(match[1], ".")[0]
		if err := add(root, "string", match[1]); err != nil {
			return nil, err
		}
	}
	for _, match := range plainServerValue.FindAllStringSubmatch(source, -1) {
		if err := add(match[1], "string", match[1]); err != nil {
			return nil, err
		}
	}
	for _, match := range serverIf.FindAllStringSubmatch(source, -1) {
		root := strings.Split(match[1], ".")[0]
		if err := add(root, "bool", match[1]); err != nil {
			return nil, err
		}
	}
	if match := serverFor.FindStringSubmatch(source); match != nil {
		return nil, fmt.Errorf("%s cannot infer the element type of %s; add an `interface Props { ... }` frontmatter contract", path, match[1])
	}
	for _, match := range clientPropValue.FindAllStringSubmatch(source, -1) {
		expression := match[1] + match[2]
		if err := add(match[1], "string", expression); err != nil {
			return nil, err
		}
	}
	names := make([]string, 0, len(types))
	for name := range types {
		names = append(names, name)
	}
	sort.Strings(names)
	props := make([]prop, 0, len(names))
	for _, name := range names {
		props = append(props, prop{Name: name, Type: types[name]})
	}
	return props, nil
}

func generateRoutePropFiles(views []routeView) (map[string][]byte, error) {
	type group struct {
		packageName string
		views       []routeView
	}
	groups := map[string]*group{}
	for _, view := range views {
		if !view.GeneratedProps {
			continue
		}
		current := groups[view.SourceDir]
		if current == nil {
			current = &group{packageName: view.PackageName}
			groups[view.SourceDir] = current
		}
		if current.packageName != view.PackageName {
			return nil, fmt.Errorf("route package mismatch in %s", view.SourceDir)
		}
		current.views = append(current.views, view)
	}
	files := map[string][]byte{}
	for directory, current := range groups {
		generated, err := generateRoutePropFile(current.packageName, current.views)
		if err != nil {
			return nil, fmt.Errorf("generate route props in %s: %w", directory, err)
		}
		relative := filepath.Base(directory)
		if len(current.views) > 0 {
			relative = current.views[0].Directory
		}
		if relative == "." {
			relative = "root"
		}
		files[filepath.Join(filepath.FromSlash(relative), routePropsFilename)] = generated
	}
	return files, nil
}

func generateRoutePropFile(packageName string, views []routeView) ([]byte, error) {
	imports := map[string]string{}
	for _, view := range views {
		for _, imported := range view.Imports {
			if existing := imports[imported.Alias]; existing != "" && existing != imported.Path {
				return nil, fmt.Errorf("import alias %s refers to both %s and %s", imported.Alias, existing, imported.Path)
			}
			imports[imported.Alias] = imported.Path
		}
	}
	aliases := make([]string, 0, len(imports))
	for alias := range imports {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	sort.Slice(views, func(i, j int) bool { return views[i].Kind < views[j].Kind })

	var output bytes.Buffer
	output.WriteString("// Code generated by Northframe. DO NOT EDIT.\n\n")
	fmt.Fprintf(&output, "package %s\n", packageName)
	if len(aliases) > 0 {
		output.WriteString("\nimport (\n")
		for _, alias := range aliases {
			fmt.Fprintf(&output, "\t%s %s\n", alias, strconv.Quote(imports[alias]))
		}
		output.WriteString(")\n")
	}
	for _, view := range views {
		fmt.Fprintf(&output, "\ntype %s struct {\n", sidecarPropsName(view.SourcePath))
		for _, field := range view.Props {
			fmt.Fprintf(&output, "\t%s %s\n", field.Name, field.Type)
		}
		output.WriteString("}\n")
	}
	formatted, err := format.Source(output.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format generated route props: %w\n%s", err, output.String())
	}
	return formatted, nil
}
