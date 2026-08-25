package main

import (
	"context"
	"fmt"
	"os"
	"sort"

	"northframe.dev/northframe/internal/dependencies"
)

func addDependency(arguments []string) error {
	manager, manifest, err := dependencyManager()
	if err != nil {
		return err
	}
	lock, err := manager.Add(context.Background(), arguments)
	if err != nil {
		return err
	}
	updated, err := dependencies.LoadManifest(manager.Root)
	if err != nil {
		return err
	}
	printDependencyChanges("added", manifest, updated, lock)
	return nil
}

func removeDependency(arguments []string) error {
	manager, manifest, err := dependencyManager()
	if err != nil {
		return err
	}
	lock, err := manager.Remove(context.Background(), arguments)
	if err != nil {
		return err
	}
	updated, err := dependencies.LoadManifest(manager.Root)
	if err != nil {
		return err
	}
	printDependencyChanges("removed", manifest, updated, lock)
	return nil
}

func updateDependencies(arguments []string) error {
	if len(arguments) != 0 {
		return fmt.Errorf("update does not accept package names yet; edit northframe.toml or run `north add package@version`")
	}
	manager, _, err := dependencyManager()
	if err != nil {
		return err
	}
	lock, err := manager.Update(context.Background())
	if err != nil {
		return err
	}
	fmt.Printf("resolved %d JavaScript package(s) into northframe.lock\n", len(lock.Packages))
	return nil
}

func dependencyManager() (dependencies.Manager, dependencies.Manifest, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return dependencies.Manager{}, dependencies.Manifest{}, err
	}
	root, err := dependencies.FindProjectRoot(workingDirectory)
	if err != nil {
		return dependencies.Manager{}, dependencies.Manifest{}, err
	}
	manifest, err := dependencies.LoadManifest(root)
	if err != nil {
		return dependencies.Manager{}, dependencies.Manifest{}, err
	}
	return dependencies.Manager{Root: root, Registry: os.Getenv("NORTHFRAME_NPM_REGISTRY")}, manifest, nil
}

func printDependencyChanges(action string, before, after dependencies.Manifest, lock dependencies.Lockfile) {
	var names []string
	if action == "removed" {
		for name := range before.Client.Dependencies {
			if _, exists := after.Client.Dependencies[name]; !exists {
				names = append(names, name)
			}
		}
	} else {
		for name := range after.Client.Dependencies {
			if before.Client.Dependencies[name] != after.Client.Dependencies[name] {
				names = append(names, name)
			}
		}
	}
	sort.Strings(names)
	for _, name := range names {
		if current, exists := lock.Packages[name]; exists {
			fmt.Printf("%s %s@%s\n", action, name, current.Version)
		} else {
			fmt.Printf("%s %s\n", action, name)
		}
	}
	fmt.Printf("resolved %d package(s); no package.json or Node installation required\n", len(lock.Packages))
}
