package dependencies

import (
	"fmt"
	"path/filepath"
	"strings"
)

type PackageSpec struct {
	Name       string
	Constraint string
}

func ParsePackageSpec(value string) (PackageSpec, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return PackageSpec{}, fmt.Errorf("package name cannot be empty")
	}
	name, constraint := value, "latest"
	if strings.HasPrefix(value, "@") {
		slash := strings.Index(value, "/")
		if slash < 2 {
			return PackageSpec{}, fmt.Errorf("invalid scoped package %q", value)
		}
		if separator := strings.LastIndex(value, "@"); separator > slash {
			name, constraint = value[:separator], value[separator+1:]
		}
	} else if separator := strings.LastIndex(value, "@"); separator > 0 {
		name, constraint = value[:separator], value[separator+1:]
	}
	if !validPackageName(name) {
		return PackageSpec{}, fmt.Errorf("invalid package name %q", name)
	}
	if strings.TrimSpace(constraint) == "" {
		return PackageSpec{}, fmt.Errorf("package %s has an empty version", name)
	}
	return PackageSpec{Name: name, Constraint: constraint}, nil
}

func PackageName(importPath string) string {
	if strings.HasPrefix(importPath, "@") {
		parts := strings.Split(importPath, "/")
		if len(parts) >= 2 {
			return strings.Join(parts[:2], "/")
		}
	}
	if part, _, found := strings.Cut(importPath, "/"); found {
		return part
	}
	return importPath
}

func packageDirectory(nodeModules, name string) string {
	return filepath.Join(nodeModules, filepath.FromSlash(name))
}

func validPackageName(name string) bool {
	if name == "" || strings.ContainsAny(name, "\\ :#") || strings.Contains(name, "..") {
		return false
	}
	if strings.HasPrefix(name, "@") {
		parts := strings.Split(name, "/")
		return len(parts) == 2 && len(parts[0]) > 1 && parts[1] != ""
	}
	return !strings.Contains(name, "/") && !strings.HasPrefix(name, ".")
}
