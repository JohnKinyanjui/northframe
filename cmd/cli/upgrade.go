package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func upgrade(arguments []string) error {
	flags := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	options := addProjectFlags(flags, true)
	check := flags.Bool("check", false, "show managed changes without writing them")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("upgrade does not accept positional arguments")
	}

	build, err := compileProject(*options)
	if err != nil {
		return fmt.Errorf("upgrade preflight failed without changing the app: %w", err)
	}
	changes, err := generatedChanges(options.generated, build.Files)
	if err != nil {
		return err
	}
	sort.Strings(changes)
	printUpgradePlan(*check, changes)
	if *check {
		return nil
	}

	previous, err := readGeneratedFiles(options.generated)
	if err != nil {
		return err
	}
	if err := writeGeneratedFiles(options.generated, build.Files); err != nil {
		return err
	}
	validationDirectory, err := os.MkdirTemp("", "north-upgrade-")
	if err != nil {
		_ = restoreUpgradeFiles(options.generated, previous)
		return err
	}
	defer os.RemoveAll(validationDirectory)
	if err := buildBinary(options.target, filepath.Join(validationDirectory, "app")); err != nil {
		if restoreErr := restoreUpgradeFiles(options.generated, previous); restoreErr != nil {
			return fmt.Errorf("upgrade validation failed (%v) and restoring generated files failed: %w", err, restoreErr)
		}
		return fmt.Errorf("upgrade validation failed; restored the previous generated files: %w", err)
	}

	fmt.Printf("upgraded Northframe-managed files for %d route(s)\n", build.RouteCount)
	fmt.Println("handwritten application sources, database files, and dependency versions were left unchanged")
	return nil
}

func restoreUpgradeFiles(generated string, files map[string][]byte) error {
	return writeGeneratedFiles(generated, files)
}

func generatedChanges(directory string, desired map[string][]byte) ([]string, error) {
	current, err := readGeneratedFiles(directory)
	if err != nil {
		return nil, err
	}
	changes := make([]string, 0)
	for name, contents := range desired {
		if existing, found := current[name]; !found {
			changes = append(changes, "create "+name)
		} else if !bytes.Equal(existing, contents) {
			changes = append(changes, "refresh "+name)
		}
	}
	for name := range current {
		if _, keep := desired[name]; !keep {
			changes = append(changes, "remove "+name)
		}
	}
	sort.Strings(changes)
	return changes, nil
}

func readGeneratedFiles(directory string) (map[string][]byte, error) {
	files := map[string][]byte{}
	_, err := os.Stat(directory)
	if errors.Is(err, os.ErrNotExist) {
		return files, nil
	}
	if err != nil {
		return nil, err
	}
	err = filepath.WalkDir(directory, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !isGeneratedSource(entry.Name()) {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}
		files[relative] = contents
		return nil
	})
	return files, err
}

func printUpgradePlan(check bool, changes []string) {
	if len(changes) == 0 {
		fmt.Println("Northframe-managed generated files are already current")
		return
	}
	verb := "will"
	if check {
		verb = "would"
	}
	fmt.Printf("Northframe %s change %d managed file(s):\n", verb, len(changes))
	for _, change := range changes {
		fmt.Println("  -", change)
	}
}
