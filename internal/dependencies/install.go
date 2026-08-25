package dependencies

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	semver "github.com/Masterminds/semver/v3"
)

type Manager struct {
	Root     string
	Registry string
	Client   *http.Client
}

func (manager Manager) Add(ctx context.Context, values []string) (Lockfile, error) {
	if len(values) == 0 {
		return Lockfile{}, errors.New("add requires at least one package")
	}
	manifest, err := LoadManifest(manager.Root)
	if err != nil {
		return Lockfile{}, err
	}
	for _, value := range values {
		spec, err := ParsePackageSpec(value)
		if err != nil {
			return Lockfile{}, err
		}
		manifest.Client.Dependencies[spec.Name] = spec.Constraint
	}
	lock, err := manager.install(ctx, manifest)
	if err != nil {
		return Lockfile{}, err
	}
	if err := SaveManifest(manager.Root, manifest); err != nil {
		return Lockfile{}, err
	}
	if err := SaveLock(manager.Root, lock); err != nil {
		return Lockfile{}, err
	}
	return lock, nil
}

func (manager Manager) Remove(ctx context.Context, names []string) (Lockfile, error) {
	if len(names) == 0 {
		return Lockfile{}, errors.New("remove requires at least one package")
	}
	manifest, err := LoadManifest(manager.Root)
	if err != nil {
		return Lockfile{}, err
	}
	for _, raw := range names {
		spec, parseErr := ParsePackageSpec(raw)
		if parseErr != nil {
			return Lockfile{}, parseErr
		}
		if _, exists := manifest.Client.Dependencies[spec.Name]; !exists {
			return Lockfile{}, fmt.Errorf("dependency %s is not installed", spec.Name)
		}
		delete(manifest.Client.Dependencies, spec.Name)
	}
	lock, err := manager.install(ctx, manifest)
	if err != nil {
		return Lockfile{}, err
	}
	if err := SaveManifest(manager.Root, manifest); err != nil {
		return Lockfile{}, err
	}
	if err := SaveLock(manager.Root, lock); err != nil {
		return Lockfile{}, err
	}
	return lock, nil
}

func (manager Manager) Update(ctx context.Context) (Lockfile, error) {
	manifest, err := LoadManifest(manager.Root)
	if err != nil {
		return Lockfile{}, err
	}
	lock, err := manager.install(ctx, manifest)
	if err != nil {
		return Lockfile{}, err
	}
	if err := SaveManifest(manager.Root, manifest); err != nil {
		return Lockfile{}, err
	}
	if err := SaveLock(manager.Root, lock); err != nil {
		return Lockfile{}, err
	}
	return lock, nil
}

type dependencyRequest struct {
	Name        string
	Constraint  string
	RequestedBy string
}

func (manager Manager) install(ctx context.Context, manifest Manifest) (Lockfile, error) {
	registry := newRegistryClient(manager.Registry, manager.Client)
	requests := make([]dependencyRequest, 0, len(manifest.Client.Dependencies))
	for _, name := range DependencyNames(manifest) {
		requests = append(requests, dependencyRequest{Name: name, Constraint: manifest.Client.Dependencies[name], RequestedBy: ManifestName})
	}
	resolved := map[string]packageVersion{}
	for len(requests) > 0 {
		request := requests[0]
		requests = requests[1:]
		if current, exists := resolved[request.Name]; exists {
			if versionSatisfies(current.Version, request.Constraint, registry.cache[request.Name].Tags) {
				continue
			}
			return Lockfile{}, fmt.Errorf("dependency conflict: %s@%s selected, but %s requires %q", request.Name, current.Version, request.RequestedBy, request.Constraint)
		}
		selected, err := registry.resolve(ctx, request.Name, request.Constraint)
		if err != nil {
			return Lockfile{}, err
		}
		if selected.HasInstallScript {
			return Lockfile{}, fmt.Errorf("%s@%s requires an install script, which Northframe does not execute", request.Name, selected.Version)
		}
		resolved[request.Name] = selected
		for name, version := range selected.Dependencies {
			requests = append(requests, dependencyRequest{Name: name, Constraint: version, RequestedBy: request.Name + "@" + selected.Version})
		}
		for name, version := range selected.PeerDependencies {
			if selected.PeerDependenciesMeta[name].Optional {
				continue
			}
			requests = append(requests, dependencyRequest{Name: name, Constraint: version, RequestedBy: request.Name + "@" + selected.Version + " peer"})
		}
	}

	lock := Lockfile{Version: lockVersion, Packages: make(map[string]LockedPackage, len(resolved))}
	for name, selected := range resolved {
		integrity := selected.Dist.Integrity
		if integrity == "" && selected.Dist.Shasum != "" {
			integrity = "sha1-" + selected.Dist.Shasum
		}
		if selected.Dist.Tarball == "" || integrity == "" {
			return Lockfile{}, fmt.Errorf("registry metadata for %s@%s has no verifiable tarball", name, selected.Version)
		}
		lock.Packages[name] = LockedPackage{
			Version: selected.Version, Resolved: selected.Dist.Tarball, Integrity: integrity,
			Dependencies: mergeRuntimeDependencies(selected),
		}
	}
	for name, constraint := range manifest.Client.Dependencies {
		locked := lock.Packages[name]
		locked.Constraint = constraint
		lock.Packages[name] = locked
	}
	if err := manager.materialize(ctx, lock); err != nil {
		return Lockfile{}, err
	}
	return lock, nil
}

func (manager Manager) materialize(ctx context.Context, lock Lockfile) error {
	storeRoot := filepath.Join(manager.Root, ".northframe")
	if err := os.MkdirAll(storeRoot, 0o755); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(storeRoot, "modules-staging-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	nodeModules := filepath.Join(staging, "node_modules")
	if err := os.MkdirAll(nodeModules, 0o755); err != nil {
		return err
	}
	names := make([]string, 0, len(lock.Packages))
	for name := range lock.Packages {
		names = append(names, name)
	}
	sort.Strings(names)
	client := manager.Client
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	for _, name := range names {
		current := lock.Packages[name]
		if err := downloadAndExtract(ctx, client, current.Resolved, current.Integrity, packageDirectory(nodeModules, name)); err != nil {
			return fmt.Errorf("install %s@%s: %w", name, current.Version, err)
		}
	}
	final := filepath.Join(storeRoot, "modules")
	backup := filepath.Join(storeRoot, "modules-backup")
	_ = os.RemoveAll(backup)
	if _, err := os.Stat(final); err == nil {
		if err := os.Rename(final, backup); err != nil {
			return err
		}
	}
	if err := os.Rename(staging, final); err != nil {
		_ = os.Rename(backup, final)
		return err
	}
	_ = os.RemoveAll(backup)
	return nil
}

func mergeRuntimeDependencies(version packageVersion) map[string]string {
	result := cloneStrings(version.Dependencies)
	for name, constraint := range version.PeerDependencies {
		if !version.PeerDependenciesMeta[name].Optional {
			result[name] = constraint
		}
	}
	return result
}

func versionSatisfies(version, constraint string, tags map[string]string) bool {
	if tagged := tags[constraint]; tagged != "" {
		return version == tagged
	}
	return semverSatisfies(version, constraint)
}

func semverSatisfies(version, constraint string) bool {
	current, err := semver.NewVersion(version)
	if err != nil {
		return false
	}
	rule, err := semver.NewConstraint(constraint)
	return err == nil && rule.Check(current)
}
