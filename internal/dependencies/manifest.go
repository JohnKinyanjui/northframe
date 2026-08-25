package dependencies

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const (
	ManifestName = "northframe.toml"
	LockName     = "northframe.lock"
	lockVersion  = 1
)

type Manifest struct {
	Client ClientConfig `toml:"client"`
}

type ClientConfig struct {
	Source       string            `toml:"source"`
	Dependencies map[string]string `toml:"dependencies"`
}

type Lockfile struct {
	Version  int                      `json:"version"`
	Packages map[string]LockedPackage `json:"packages"`
}

type LockedPackage struct {
	Version      string            `json:"version"`
	Constraint   string            `json:"constraint,omitempty"`
	Resolved     string            `json:"resolved"`
	Integrity    string            `json:"integrity"`
	Dependencies map[string]string `json:"dependencies,omitempty"`
}

type Project struct {
	Root         string
	ClientSource string
	NodeModules  string
	Dependencies map[string]string
}

func DefaultManifest() Manifest {
	return Manifest{Client: ClientConfig{Source: "web/client", Dependencies: map[string]string{}}}
}

func LoadManifest(root string) (Manifest, error) {
	manifest := DefaultManifest()
	contents, err := os.ReadFile(filepath.Join(root, ManifestName))
	if errors.Is(err, os.ErrNotExist) {
		return manifest, nil
	}
	if err != nil {
		return Manifest{}, fmt.Errorf("read %s: %w", ManifestName, err)
	}
	if err := toml.Unmarshal(contents, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse %s: %w", ManifestName, err)
	}
	if strings.TrimSpace(manifest.Client.Source) == "" {
		manifest.Client.Source = "web/client"
	}
	if manifest.Client.Dependencies == nil {
		manifest.Client.Dependencies = map[string]string{}
	}
	return manifest, nil
}

func SaveManifest(root string, manifest Manifest) error {
	if strings.TrimSpace(manifest.Client.Source) == "" {
		manifest.Client.Source = "web/client"
	}
	if manifest.Client.Dependencies == nil {
		manifest.Client.Dependencies = map[string]string{}
	}
	contents, err := toml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("encode %s: %w", ManifestName, err)
	}
	return writeAtomic(filepath.Join(root, ManifestName), contents)
}

func LoadLock(root string) (Lockfile, error) {
	contents, err := os.ReadFile(filepath.Join(root, LockName))
	if err != nil {
		return Lockfile{}, err
	}
	var lock Lockfile
	if err := json.Unmarshal(contents, &lock); err != nil {
		return Lockfile{}, fmt.Errorf("parse %s: %w", LockName, err)
	}
	if lock.Version != lockVersion {
		return Lockfile{}, fmt.Errorf("unsupported %s format %d; run `north update`", LockName, lock.Version)
	}
	if lock.Packages == nil {
		lock.Packages = map[string]LockedPackage{}
	}
	return lock, nil
}

func SaveLock(root string, lock Lockfile) error {
	lock.Version = lockVersion
	if lock.Packages == nil {
		lock.Packages = map[string]LockedPackage{}
	}
	contents, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", LockName, err)
	}
	contents = append(contents, '\n')
	return writeAtomic(filepath.Join(root, LockName), contents)
}

func FindProjectRoot(start string) (string, error) {
	absolute, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	info, statErr := os.Stat(absolute)
	if statErr == nil && !info.IsDir() {
		absolute = filepath.Dir(absolute)
	}
	for current := absolute; ; current = filepath.Dir(current) {
		if fileExists(filepath.Join(current, ManifestName)) || applicationRoot(current) || fileExists(filepath.Join(current, "go.mod")) {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("cannot find a Northframe project above %s", start)
		}
	}
}

func applicationRoot(directory string) bool {
	routes, err := os.Stat(filepath.Join(directory, "web", "routes"))
	return err == nil && routes.IsDir() && fileExists(filepath.Join(directory, "main.go"))
}

func InspectProject(start string) (Project, error) {
	root, err := FindProjectRoot(start)
	if err != nil {
		return Project{}, err
	}
	manifest, err := LoadManifest(root)
	if err != nil {
		return Project{}, err
	}
	project := Project{
		Root: root, ClientSource: filepath.Join(root, filepath.FromSlash(manifest.Client.Source)),
		NodeModules:  filepath.Join(root, ".northframe", "modules", "node_modules"),
		Dependencies: cloneStrings(manifest.Client.Dependencies),
	}
	if len(project.Dependencies) == 0 {
		return project, nil
	}
	lock, err := LoadLock(root)
	if errors.Is(err, os.ErrNotExist) {
		return Project{}, fmt.Errorf("%s is missing; run `north update`", LockName)
	}
	if err != nil {
		return Project{}, err
	}
	for name, constraint := range project.Dependencies {
		locked, ok := lock.Packages[name]
		if !ok {
			return Project{}, fmt.Errorf("dependency %s is not locked; run `north update`", name)
		}
		if locked.Constraint != constraint {
			return Project{}, fmt.Errorf("dependency %s lock uses %q instead of %q; run `north update`", name, locked.Constraint, constraint)
		}
	}
	for name := range lock.Packages {
		if !fileExists(filepath.Join(packageDirectory(project.NodeModules, name), "package.json")) {
			return Project{}, fmt.Errorf("dependency files for %s are missing; run `north update`", name)
		}
	}
	return project, nil
}

func DependencyNames(manifest Manifest) []string {
	names := make([]string, 0, len(manifest.Client.Dependencies))
	for name := range manifest.Client.Dependencies {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func writeAtomic(path string, contents []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".northframe-write-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o644); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(contents); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func cloneStrings(input map[string]string) map[string]string {
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}
