package compiler

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	pathpkg "path"
	"strconv"
)

type sidecarInfo struct {
	HasActions    bool
	HasMiddleware bool
	Props         []prop
	Imports       []componentImport
	PackageName   string
	DeclaresProps bool
}

func inspectSidecar(path, kind string) (sidecarInfo, error) {
	source, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return sidecarInfo{}, fmt.Errorf("%s is required beside %s.north", path, kind)
		}
		return sidecarInfo{}, fmt.Errorf("parse %s: %w", path, err)
	}
	names := sidecarNamesFor(kind)
	loaderFound := false
	loaderValid := false
	info := sidecarInfo{PackageName: source.Name.Name}
	for _, declaration := range source.Decls {
		switch current := declaration.(type) {
		case *ast.GenDecl:
			for _, specification := range current.Specs {
				if typeSpec, ok := specification.(*ast.TypeSpec); ok && typeSpec.Name.Name == names.props {
					info.DeclaresProps = true
					structure, ok := typeSpec.Type.(*ast.StructType)
					if !ok {
						return sidecarInfo{}, fmt.Errorf("%s type %s must be a struct", path, names.props)
					}
					info.Props, err = sidecarProps(structure)
					if err != nil {
						return sidecarInfo{}, fmt.Errorf("%s type %s: %w", path, names.props, err)
					}
				}
			}
		case *ast.FuncDecl:
			switch current.Name.Name {
			case names.loader:
				loaderFound = true
				loaderValid = validLoaderSignature(current, names.props)
			case names.actions:
				info.HasActions = names.actions != ""
			case names.middleware:
				info.HasMiddleware = true
			}
		}
	}
	for _, specification := range source.Imports {
		importPath, unquoteErr := strconv.Unquote(specification.Path.Value)
		if unquoteErr != nil {
			return sidecarInfo{}, fmt.Errorf("parse import %s: %w", specification.Path.Value, unquoteErr)
		}
		alias := pathpkg.Base(importPath)
		if specification.Name != nil {
			alias = specification.Name.Name
		}
		if alias != "_" && alias != "." {
			info.Imports = append(info.Imports, componentImport{Alias: alias, Path: importPath})
		}
	}
	if !loaderFound || !loaderValid {
		return sidecarInfo{}, fmt.Errorf("%s must declare func %s(*web.Context) (%s, error)", path, names.loader, names.props)
	}
	return info, nil
}

func sidecarProps(structure *ast.StructType) ([]prop, error) {
	var result []prop
	for _, field := range structure.Fields.List {
		if len(field.Names) == 0 {
			return nil, fmt.Errorf("embedded fields are not supported in client contracts")
		}
		var typeSource bytes.Buffer
		if err := format.Node(&typeSource, token.NewFileSet(), field.Type); err != nil {
			return nil, err
		}
		for _, name := range field.Names {
			if !name.IsExported() {
				continue
			}
			result = append(result, prop{Name: name.Name, Type: typeSource.String()})
		}
	}
	return result, nil
}

type sidecarNames struct {
	loader     string
	props      string
	actions    string
	middleware string
}

func sidecarNamesFor(kind string) sidecarNames {
	if kind == "layout" {
		return sidecarNames{loader: "Layout", props: "LayoutProps", middleware: "LayoutMiddleware"}
	}
	return sidecarNames{loader: "Page", props: "PageProps", actions: "PageActions", middleware: "PageMiddleware"}
}

func validLoaderSignature(function *ast.FuncDecl, propsName string) bool {
	if function.Recv != nil || function.Type.Params == nil || function.Type.Results == nil {
		return false
	}
	if function.Type.Params.NumFields() != 1 || function.Type.Results.NumFields() != 2 {
		return false
	}
	parameter, parameterOK := function.Type.Params.List[0].Type.(*ast.StarExpr)
	selector, selectorOK := parameter.X.(*ast.SelectorExpr)
	firstName := ""
	switch first := function.Type.Results.List[0].Type.(type) {
	case *ast.Ident:
		firstName = first.Name
	case *ast.SelectorExpr:
		firstName = first.Sel.Name
	}
	second, secondOK := function.Type.Results.List[1].Type.(*ast.Ident)
	return parameterOK && selectorOK && selector.Sel.Name == "Context" &&
		firstName == propsName && secondOK && second.Name == "error"
}
