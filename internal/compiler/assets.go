package compiler

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type publicAsset struct {
	Path        string
	Content     []byte
	ContentType string
}

func discoverPublicAssets(root string) ([]publicAsset, error) {
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("inspect public directory: %w", err)
	}
	var assets []publicAsset
	err := filepath.WalkDir(root, func(assetPath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		content, err := os.ReadFile(assetPath)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, assetPath)
		if err != nil {
			return err
		}
		assets = append(assets, publicAsset{
			Path: filepath.ToSlash(relative), Content: content,
			ContentType: publicContentType(filepath.Ext(assetPath)),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read public assets: %w", err)
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].Path < assets[j].Path })
	return assets, nil
}

func publicContentType(extension string) string {
	switch strings.ToLower(extension) {
	case ".webmanifest":
		return "application/manifest+json"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	}
	if contentType := mime.TypeByExtension(extension); contentType != "" {
		return contentType
	}
	return "application/octet-stream"
}
