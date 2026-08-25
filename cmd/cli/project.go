package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"northframe.dev/northframe/internal/compiler"
)

type projectOptions struct {
	routes      string
	api         string
	generated   string
	packageName string
	routeImport string
	apiImport   string
	target      string
}

func addProjectFlags(flags *flag.FlagSet, includeTarget bool) *projectOptions {
	options := &projectOptions{}
	flags.StringVar(&options.routes, "routes", "web/routes", "structured routes directory")
	flags.StringVar(&options.api, "api", "", "API routes directory (defaults to <routes>/api)")
	flags.StringVar(&options.generated, "generated", ".generated/routes", "protected generated Go package directory")
	flags.StringVar(&options.packageName, "package", "routes", "generated Go package name")
	flags.StringVar(&options.routeImport, "route-import", "", "Go import path for the routes directory (detected from go.mod)")
	flags.StringVar(&options.apiImport, "api-import", "", "Go import path for the API directory (detected from go.mod)")
	if includeTarget {
		flags.StringVar(&options.target, "target", ".", "Go main package")
	}
	return options
}

func generateProject(options projectOptions) (int, error) {
	build, err := compileProject(options)
	if err != nil {
		return 0, err
	}
	if err := writeGeneratedFiles(options.generated, build.Files); err != nil {
		return 0, err
	}
	return build.RouteCount, nil
}

func compileProject(options projectOptions) (compiler.RouteBuild, error) {
	if err := validateRoutesDirectory(options.routes); err != nil {
		return compiler.RouteBuild{}, err
	}
	routeImport, err := resolveRouteImport(options.routes, options.routeImport)
	if err != nil {
		return compiler.RouteBuild{}, err
	}
	apiDirectory := options.api
	if strings.TrimSpace(apiDirectory) == "" {
		apiDirectory = filepath.Join(options.routes, "api")
	}
	apiImport, err := optionalImportPath(apiDirectory, options.apiImport)
	if err != nil {
		return compiler.RouteBuild{}, err
	}
	generatedImport, err := resolveRouteImport(options.generated, "")
	if err != nil {
		return compiler.RouteBuild{}, fmt.Errorf("resolve generated import path: %w", err)
	}
	build, err := compiler.BuildProjectTo(options.routes, apiDirectory, options.packageName, routeImport, apiImport, generatedImport)
	if err != nil {
		return compiler.RouteBuild{}, err
	}
	return build, nil
}

func optionalImportPath(directory, configured string) (string, error) {
	info, err := os.Stat(directory)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("inspect API directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("API path %q is not a directory", directory)
	}
	return resolveRouteImport(directory, configured)
}

func validateRoutesDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			workingDirectory, _ := os.Getwd()
			if filepath.Clean(path) == filepath.Clean("web/routes") {
				if legacy, legacyErr := os.Stat("routes"); legacyErr == nil && legacy.IsDir() {
					return fmt.Errorf("legacy routes directory found at %s; move routes, components, client, and public below web/ or temporarily pass -routes routes", filepath.Join(workingDirectory, "routes"))
				}
			}
			return fmt.Errorf("routes directory %q not found in %s; run this command from the application root or pass -routes", path, workingDirectory)
		}
		return fmt.Errorf("inspect routes directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("routes path %q is not a directory", path)
	}
	return nil
}

func writeGeneratedFiles(directory string, files map[string][]byte) error {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create generated directory: %w", err)
	}
	desired := make(map[string]struct{}, len(files))
	for name := range files {
		desired[filepath.Clean(name)] = struct{}{}
	}
	if err := filepath.WalkDir(directory, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}
		if !isGeneratedSource(entry.Name()) {
			return nil
		}
		if _, keep := desired[filepath.Clean(relative)]; keep {
			return nil
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove stale generated file: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		path := filepath.Join(directory, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create generated route directory: %w", err)
		}
		if err := atomicWriteFile(path, files[name], 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}
	return nil
}

func atomicWriteFile(path string, contents []byte, mode os.FileMode) (writeErr error) {
	temporary, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}()
	if err := temporary.Chmod(mode); err != nil {
		return err
	}
	if _, err := temporary.Write(contents); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func isGeneratedSource(name string) bool {
	return strings.HasSuffix(name, "_generated.go") || strings.HasSuffix(name, "_northframe.go") || strings.HasSuffix(name, "_generated.ts")
}

func resolveRouteImport(routesDirectory, configured string) (string, error) {
	if configured != "" {
		return strings.TrimRight(configured, "/"), nil
	}
	absoluteRoutes, err := filepath.Abs(routesDirectory)
	if err != nil {
		return "", fmt.Errorf("resolve routes directory: %w", err)
	}
	current := absoluteRoutes
	for {
		modulePath, found, err := modulePathAt(current)
		if err != nil {
			return "", err
		}
		if found {
			relative, relErr := filepath.Rel(current, absoluteRoutes)
			if relErr != nil || strings.HasPrefix(relative, "..") {
				return "", fmt.Errorf("routes directory is outside the Go module")
			}
			if relative == "." {
				return modulePath, nil
			}
			return strings.TrimRight(modulePath, "/") + "/" + filepath.ToSlash(relative), nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", errors.New("cannot determine route import path: no go.mod found; pass -route-import")
}

func modulePathAt(directory string) (string, bool, error) {
	goModPath := filepath.Join(directory, "go.mod")
	contents, err := os.ReadFile(goModPath)
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read %s: %w", goModPath, err)
	}
	for _, line := range strings.Split(string(contents), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), true, nil
		}
	}
	return "", false, fmt.Errorf("%s does not declare a module path", goModPath)
}
