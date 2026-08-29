package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/JohnKinyanjui/northframe/internal/dependencies"
)

var errInterrupted = errors.New("interrupted")

func waitForChange(root, generated, baseline string, interrupts <-chan os.Signal) error {
	ticker := time.NewTicker(450 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-interrupts:
			return errInterrupted
		case <-ticker.C:
			current, err := watchSignature(root, generated)
			if err != nil {
				return err
			}
			if current != baseline {
				return nil
			}
		}
	}
}

func watchSignature(root, generated string) (string, error) {
	absoluteGenerated, _ := filepath.Abs(generated)
	var records []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		absolutePath, _ := filepath.Abs(path)
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == ".northframe" || entry.Name() == ".generated" || absolutePath == absoluteGenerated {
				return filepath.SkipDir
			}
			return nil
		}
		extension := filepath.Ext(path)
		relative, _ := filepath.Rel(root, path)
		inPublic := strings.HasPrefix(filepath.ToSlash(relative), "web/public/")
		if !inPublic && extension != ".go" && extension != ".north" && extension != ".css" && extension != ".ts" && extension != ".tsx" && extension != ".js" && extension != ".jsx" && extension != ".json" && entry.Name() != ".env" && entry.Name() != "go.mod" && entry.Name() != "go.sum" && entry.Name() != dependencies.ManifestName && entry.Name() != dependencies.LockName {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		records = append(records, fmt.Sprintf("%s:%d:%d", path, info.ModTime().UnixNano(), info.Size()))
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(records)
	return strings.Join(records, "|"), nil
}
