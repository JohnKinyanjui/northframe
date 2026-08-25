package compiler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type typeContractResolver struct {
	modulePath string
	moduleRoot string
}

type typeContext struct {
	directory string
	imports   map[string]string
}

func newTypeContractResolver(start string) (*typeContractResolver, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return nil, err
	}
	for {
		contents, readErr := os.ReadFile(filepath.Join(current, "go.mod"))
		if readErr == nil {
			for _, line := range strings.Split(string(contents), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "module ") {
					return &typeContractResolver{modulePath: strings.TrimSpace(strings.TrimPrefix(line, "module ")), moduleRoot: current}, nil
				}
			}
			return nil, fmt.Errorf("%s does not declare a module", filepath.Join(current, "go.mod"))
		}
		if !os.IsNotExist(readErr) {
			return nil, readErr
		}
		parent := filepath.Dir(current)
		if parent == current {
			absoluteStart, _ := filepath.Abs(start)
			return &typeContractResolver{moduleRoot: absoluteStart}, nil
		}
		current = parent
	}
}

func (resolver *typeContractResolver) contract(name string, props []prop, directory string, imports []componentImport) string {
	context := typeContext{directory: directory, imports: importMap(imports)}
	var output strings.Builder
	fmt.Fprintf(&output, "type %s = {\n", name)
	for _, field := range props {
		fmt.Fprintf(&output, "  readonly %s: %s;\n", field.Name, resolver.resolveSource(field.Type, context, map[string]bool{}))
	}
	output.WriteString("};")
	return output.String()
}

func (resolver *typeContractResolver) resolveSource(source string, context typeContext, seen map[string]bool) string {
	expression, err := parser.ParseExpr(source)
	if err != nil {
		return "unknown"
	}
	return resolver.resolveExpression(expression, context, seen)
}

func (resolver *typeContractResolver) resolveExpression(expression ast.Expr, context typeContext, seen map[string]bool) string {
	switch current := expression.(type) {
	case *ast.Ident:
		if primitive := primitiveTypeScriptType(current.Name); primitive != "" {
			return primitive
		}
		return resolver.resolveNamed(context.directory, current.Name, seen)
	case *ast.StarExpr:
		return resolver.resolveExpression(current.X, context, seen) + " | null"
	case *ast.ArrayType:
		return "ReadonlyArray<" + resolver.resolveExpression(current.Elt, context, seen) + ">"
	case *ast.MapType:
		if key := resolver.resolveExpression(current.Key, context, seen); key == "string" {
			return "Readonly<Record<string, " + resolver.resolveExpression(current.Value, context, seen) + ">>"
		}
		return "unknown"
	case *ast.InterfaceType:
		return "unknown"
	case *ast.SelectorExpr:
		alias, ok := current.X.(*ast.Ident)
		if !ok {
			return "unknown"
		}
		if alias.Name == "time" && current.Sel.Name == "Time" {
			return "string"
		}
		importPath := context.imports[alias.Name]
		directory, ok := resolver.importDirectory(importPath)
		if !ok {
			return "unknown"
		}
		return resolver.resolveNamed(directory, current.Sel.Name, seen)
	case *ast.StructType:
		return resolver.resolveStruct(current, context, seen)
	default:
		return "unknown"
	}
}

func (resolver *typeContractResolver) resolveNamed(directory, name string, seen map[string]bool) string {
	key := directory + ":" + name
	if seen[key] {
		return "unknown"
	}
	seen[key] = true
	defer delete(seen, key)
	expression, context, ok := findNamedType(directory, name)
	if !ok {
		return "unknown"
	}
	return resolver.resolveExpression(expression, context, seen)
}

func (resolver *typeContractResolver) resolveStruct(structure *ast.StructType, context typeContext, seen map[string]bool) string {
	var fields []string
	for _, field := range structure.Fields.List {
		if len(field.Names) == 0 || !field.Names[0].IsExported() {
			continue
		}
		name := field.Names[0].Name
		if field.Tag != nil {
			if tag, err := strconv.Unquote(field.Tag.Value); err == nil {
				jsonName := strings.Split(tagValue(tag, "json"), ",")[0]
				if jsonName == "-" {
					continue
				}
				if jsonName != "" {
					name = jsonName
				}
			}
		}
		fields = append(fields, fmt.Sprintf("readonly %s: %s;", name, resolver.resolveExpression(field.Type, context, seen)))
	}
	if len(fields) == 0 {
		return "Readonly<Record<string, never>>"
	}
	return "Readonly<{ " + strings.Join(fields, " ") + " }>"
}

func findNamedType(directory, name string) (ast.Expr, typeContext, bool) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, typeContext{}, false
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(directory, entry.Name()), nil, 0)
		if err != nil {
			continue
		}
		imports := make([]componentImport, 0, len(file.Imports))
		for _, specification := range file.Imports {
			path, err := strconv.Unquote(specification.Path.Value)
			if err != nil {
				continue
			}
			alias := filepath.Base(path)
			if specification.Name != nil {
				alias = specification.Name.Name
			}
			imports = append(imports, componentImport{Alias: alias, Path: path})
		}
		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, specification := range general.Specs {
				typeSpec, ok := specification.(*ast.TypeSpec)
				if ok && typeSpec.Name.Name == name {
					return typeSpec.Type, typeContext{directory: directory, imports: importMap(imports)}, true
				}
			}
		}
	}
	return nil, typeContext{}, false
}

func (resolver *typeContractResolver) importDirectory(importPath string) (string, bool) {
	if resolver.modulePath == "" {
		return "", false
	}
	if importPath == resolver.modulePath {
		return resolver.moduleRoot, true
	}
	prefix := resolver.modulePath + "/"
	if !strings.HasPrefix(importPath, prefix) {
		return "", false
	}
	return filepath.Join(resolver.moduleRoot, filepath.FromSlash(strings.TrimPrefix(importPath, prefix))), true
}

func importMap(imports []componentImport) map[string]string {
	result := make(map[string]string, len(imports))
	for _, current := range imports {
		result[current.Alias] = current.Path
	}
	return result
}

func primitiveTypeScriptType(name string) string {
	switch name {
	case "string":
		return "string"
	case "bool":
		return "boolean"
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
		return "number"
	case "any", "error":
		return "unknown"
	}
	return ""
}

func tagValue(tag, key string) string {
	for tag != "" {
		tag = strings.TrimSpace(tag)
		index := strings.IndexByte(tag, ':')
		if index < 0 {
			break
		}
		name := tag[:index]
		tag = tag[index+1:]
		if !strings.HasPrefix(tag, `"`) {
			break
		}
		end := 1
		for end < len(tag) {
			if tag[end] == '"' && tag[end-1] != '\\' {
				end++
				break
			}
			end++
		}
		quoted := tag[:end]
		value, err := strconv.Unquote(quoted)
		if err == nil && name == key {
			return value
		}
		tag = tag[end:]
	}
	return ""
}
