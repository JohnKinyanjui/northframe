package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func discoverRouteViews(root, routeImportRoot string) ([]routeView, [][]byte, error) {
	if strings.TrimSpace(routeImportRoot) == "" {
		return nil, nil, fmt.Errorf("route import root is required")
	}
	var views []routeView
	var customCSS [][]byte
	seenNames := map[string]string{}
	seenPaths := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root {
				relative, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				if filepath.ToSlash(relative) == "api" {
					return filepath.SkipDir
				}
			}
			return nil
		}
		base := filepath.Base(path)
		extension := filepath.Ext(path)
		if extension == ".nf" {
			return fmt.Errorf("%s uses the retired .nf extension; rename it to .north and rename its sidecar to .north.go", path)
		}
		if extension == ".css" {
			contents, err := readRouteCSS(path, base)
			if err != nil {
				return err
			}
			customCSS = append(customCSS, contents)
			return nil
		}
		if extension != ".north" {
			return nil
		}
		if base == "error.north" {
			if filepath.Dir(path) != root {
				return fmt.Errorf("%s: error.north currently belongs at the root of web/routes", path)
			}
			return nil
		}
		view, err := inspectRouteView(root, routeImportRoot, path, base, seenNames, seenPaths)
		if err != nil {
			return err
		}
		views = append(views, view)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	sort.Slice(views, func(i, j int) bool { return views[i].Name < views[j].Name })
	return views, customCSS, nil
}

func readRouteCSS(path, base string) ([]byte, error) {
	if base != "page.css" && base != "layout.css" {
		return nil, fmt.Errorf("%s: route CSS must be named page.css or layout.css", path)
	}
	return os.ReadFile(path)
}

func readAppCSS(routesDirectory string) ([]byte, error) {
	path := filepath.Join(filepath.Dir(routesDirectory), "app.css")
	contents, err := os.ReadFile(path)
	if err == nil {
		return contents, nil
	}
	if os.IsNotExist(err) {
		return nil, nil
	}
	return nil, fmt.Errorf("read web/app.css: %w", err)
}

func inspectRouteView(root, routeImportRoot, path, base string, seenNames, seenPaths map[string]string) (routeView, error) {
	kind := strings.TrimSuffix(base, ".north")
	if kind != "page" && kind != "layout" {
		return routeView{}, fmt.Errorf("%s: route views must be named page.north or layout.north", path)
	}
	directory, err := filepath.Rel(root, filepath.Dir(path))
	if err != nil {
		return routeView{}, err
	}
	directory = filepath.ToSlash(directory)
	name := routeComponentName(directory, kind)
	if previous := seenNames[name]; previous != "" {
		return routeView{}, fmt.Errorf("route component name %s collides between %s and %s", name, previous, path)
	}
	seenNames[name] = path
	url, err := routeURL(directory)
	if err != nil {
		return routeView{}, fmt.Errorf("%s: %w", path, err)
	}
	if kind == "page" {
		if previous := seenPaths[url]; previous != "" {
			return routeView{}, fmt.Errorf("route path %s collides between %s and %s", url, previous, path)
		}
		seenPaths[url] = path
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return routeView{}, err
	}
	sidecar, err := inspectSidecar(filepath.Join(filepath.Dir(path), kind+".north.go"), kind)
	if err != nil {
		return routeView{}, err
	}
	props, imports, generatedProps, err := resolveRouteProps(path, contents, sidecar)
	if err != nil {
		return routeView{}, err
	}
	importPath := strings.TrimRight(routeImportRoot, "/")
	if directory != "." {
		importPath += "/" + directory
	}
	propsType := name + "Props"
	contract := typeScriptContract(propsType, props)
	return routeView{
		Directory: directory, Kind: kind, Name: name, Path: url, Source: contents,
		ImportPath: importPath, ImportAlias: routeAlias(directory),
		HasActions: sidecar.HasActions, HasMiddleware: sidecar.HasMiddleware,
		Contract: contract, PropsType: propsType, Props: props, Imports: imports, SourceDir: filepath.Dir(path), SourcePath: path,
		PackageName: sidecar.PackageName, GeneratedProps: generatedProps,
	}, nil
}
