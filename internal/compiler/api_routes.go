package compiler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var apiMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "PATCH": true,
	"DELETE": true, "OPTIONS": true, "HEAD": true,
}

type apiRoute struct {
	Directory     string
	Path          string
	ImportPath    string
	ImportAlias   string
	Methods       []string
	HasMiddleware bool
}

func discoverAPIRoutes(root, importRoot string) ([]apiRoute, error) {
	if root == "" {
		return nil, nil
	}
	info, err := os.Stat(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect API directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("API path %q is not a directory", root)
	}
	if strings.TrimSpace(importRoot) == "" {
		return nil, fmt.Errorf("API import root is required when %s exists", root)
	}

	discovered := map[string]*apiRoute{}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if parseErr != nil {
			return fmt.Errorf("parse API route %s: %w", path, parseErr)
		}
		directory, relErr := filepath.Rel(root, filepath.Dir(path))
		if relErr != nil {
			return relErr
		}
		directory = filepath.ToSlash(directory)
		current := discovered[directory]
		if current == nil {
			url, routeErr := apiRouteURL(directory)
			if routeErr != nil {
				return fmt.Errorf("%s: %w", path, routeErr)
			}
			importPath := strings.TrimRight(importRoot, "/")
			if directory != "." {
				importPath += "/" + directory
			}
			current = &apiRoute{
				Directory: directory, Path: url, ImportPath: importPath, ImportAlias: apiRouteAlias(directory),
			}
			discovered[directory] = current
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv != nil {
				continue
			}
			name := function.Name.Name
			if apiMethods[name] {
				if !validAPIHandlerSignature(function) {
					return fmt.Errorf("%s func %s must have signature func %s(*web.Context) error", path, name, name)
				}
				if containsString(current.Methods, name) {
					return fmt.Errorf("%s declares duplicate %s handler for %s", path, name, current.Path)
				}
				current.Methods = append(current.Methods, name)
			}
			if name == "Middleware" {
				if !validAPIMiddlewareSignature(function) {
					return fmt.Errorf("%s func Middleware must return []web.Middleware", path)
				}
				current.HasMiddleware = true
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	routes := make([]apiRoute, 0, len(discovered))
	for _, current := range discovered {
		if len(current.Methods) == 0 {
			continue
		}
		sort.Strings(current.Methods)
		routes = append(routes, *current)
	}
	sort.Slice(routes, func(i, j int) bool { return routes[i].Path < routes[j].Path })
	return routes, nil
}

func apiRouteURL(directory string) (string, error) {
	url, err := routeURL(directory)
	if err != nil {
		return "", err
	}
	if url == "/" {
		return "/api", nil
	}
	return "/api" + url, nil
}

func apiRouteAlias(directory string) string {
	if directory == "." {
		return "apiRoot"
	}
	return "api" + exportedName(strings.ReplaceAll(directory, "/", "-"))
}

func validAPIHandlerSignature(function *ast.FuncDecl) bool {
	if function.Type.Params == nil || function.Type.Params.NumFields() != 1 || function.Type.Results == nil || function.Type.Results.NumFields() != 1 {
		return false
	}
	pointer, pointerOK := function.Type.Params.List[0].Type.(*ast.StarExpr)
	if !pointerOK {
		return false
	}
	selector, selectorOK := pointer.X.(*ast.SelectorExpr)
	result, resultOK := function.Type.Results.List[0].Type.(*ast.Ident)
	return selectorOK && selector.Sel.Name == "Context" && resultOK && result.Name == "error"
}

func validAPIMiddlewareSignature(function *ast.FuncDecl) bool {
	if function.Type.Params != nil && function.Type.Params.NumFields() != 0 || function.Type.Results == nil || function.Type.Results.NumFields() != 1 {
		return false
	}
	array, arrayOK := function.Type.Results.List[0].Type.(*ast.ArrayType)
	if !arrayOK {
		return false
	}
	selector, selectorOK := array.Elt.(*ast.SelectorExpr)
	return selectorOK && selector.Sel.Name == "Middleware"
}

func apiMethodCount(routes []apiRoute) int {
	count := 0
	for _, route := range routes {
		count += len(route.Methods)
	}
	return count
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
