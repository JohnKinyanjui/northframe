package lsp

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type resolvedGoPackage struct {
	Alias     string
	Path      string
	Directory string
}

type goPackageCacheEntry struct {
	signature string
	packages  []resolvedGoPackage
}

var goPackageCache = struct {
	sync.Mutex
	entries map[string]goPackageCacheEntry
}{entries: map[string]goPackageCacheEntry{}}

func resolvedGoPackages(uri string) []resolvedGoPackage {
	root, _ := goModuleFor(documentPath(uri))
	if root == "" {
		return nil
	}
	signature := goPackageSignature(root)
	goPackageCache.Lock()
	if cached, ok := goPackageCache.entries[root]; ok && cached.signature == signature {
		result := append([]resolvedGoPackage(nil), cached.packages...)
		goPackageCache.Unlock()
		return result
	}
	goPackageCache.Unlock()

	arguments := []string{"list", "-e", "-f", "{{if .Dir}}{{.ImportPath}}|{{.Name}}|{{.Dir}}{{end}}", "all", "std"}
	arguments = append(arguments, requiredModulePaths(root)...)
	command := exec.Command("go", arguments...)
	command.Dir = root
	output, err := command.Output()
	if err != nil && len(output) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var packages []resolvedGoPackage
	for _, raw := range strings.Split(string(output), "\n") {
		parts := strings.SplitN(strings.TrimSpace(raw), "|", 3)
		if len(parts) != 3 {
			continue
		}
		importPath, name, directory := parts[0], parts[1], parts[2]
		if importPath == "" || name == "" || name == "main" || directory == "" || seen[importPath] || inaccessibleGoPackage(importPath) {
			continue
		}
		seen[importPath] = true
		packages = append(packages, resolvedGoPackage{Alias: name, Path: importPath, Directory: directory})
	}
	sort.Slice(packages, func(i, j int) bool { return packages[i].Path < packages[j].Path })

	goPackageCache.Lock()
	goPackageCache.entries[root] = goPackageCacheEntry{signature: signature, packages: append([]resolvedGoPackage(nil), packages...)}
	goPackageCache.Unlock()
	return packages
}

func goPackageDirectory(uri, importPath string) string {
	for _, current := range resolvedGoPackages(uri) {
		if current.Path == importPath {
			return current.Directory
		}
	}
	return ""
}

func goPackageSignature(root string) string {
	var parts []string
	for _, name := range []string{"go.mod", "go.sum", "go.work"} {
		info, err := os.Stat(filepath.Join(root, name))
		if err == nil {
			parts = append(parts, name, info.ModTime().UTC().String(), strconv.FormatInt(info.Size(), 10))
		}
	}
	return strings.Join(parts, "|")
}

func requiredModulePaths(root string) []string {
	contents, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return nil
	}
	inBlock := false
	seen := map[string]bool{}
	var result []string
	for _, raw := range strings.Split(string(contents), "\n") {
		line := strings.TrimSpace(strings.SplitN(raw, "//", 2)[0])
		if line == "require (" {
			inBlock = true
			continue
		}
		if inBlock && line == ")" {
			inBlock = false
			continue
		}
		fields := strings.Fields(line)
		path := ""
		if inBlock && len(fields) >= 2 {
			path = fields[0]
		} else if len(fields) >= 3 && fields[0] == "require" {
			path = fields[1]
		}
		if path != "" && !seen[path] {
			seen[path] = true
			result = append(result, path)
		}
	}
	return result
}

func inaccessibleGoPackage(importPath string) bool {
	return strings.HasPrefix(importPath, "internal/") || strings.HasPrefix(importPath, "vendor/") || strings.Contains(importPath, "/internal/") || strings.Contains(importPath, "/vendor/")
}
