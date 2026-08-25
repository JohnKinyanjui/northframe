package dependencies

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	maxTarballSize  = 128 << 20
	maxExpandedSize = 512 << 20
	maxArchiveFiles = 100_000
)

func downloadAndExtract(ctx context.Context, client *http.Client, source, integrity, destination string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("download tarball: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download tarball: server returned %s", response.Status)
	}
	limited := io.LimitReader(response.Body, maxTarballSize+1)
	contents, err := io.ReadAll(limited)
	if err != nil {
		return err
	}
	if len(contents) > maxTarballSize {
		return fmt.Errorf("tarball exceeds %d MiB limit", maxTarballSize>>20)
	}
	if err := verifyIntegrity(contents, integrity); err != nil {
		return err
	}
	return extractPackage(contents, destination)
}

func verifyIntegrity(contents []byte, integrity string) error {
	algorithm, encoded, found := strings.Cut(strings.TrimSpace(integrity), "-")
	if !found {
		return fmt.Errorf("invalid package integrity %q", integrity)
	}
	var digest hash.Hash
	var expected []byte
	var err error
	switch algorithm {
	case "sha512":
		digest = sha512.New()
		expected, err = base64.StdEncoding.DecodeString(encoded)
	case "sha256":
		digest = sha256.New()
		expected, err = base64.StdEncoding.DecodeString(encoded)
	case "sha1":
		digest = sha1.New()
		if len(encoded) == sha1.Size*2 {
			expected, err = hex.DecodeString(encoded)
		} else {
			expected, err = base64.StdEncoding.DecodeString(encoded)
		}
	default:
		return fmt.Errorf("unsupported integrity algorithm %s", algorithm)
	}
	if err != nil {
		return fmt.Errorf("decode package integrity: %w", err)
	}
	_, _ = digest.Write(contents)
	if !equalBytes(digest.Sum(nil), expected) {
		return fmt.Errorf("package integrity check failed")
	}
	return nil
}

func extractPackage(contents []byte, destination string) error {
	gzipReader, err := gzip.NewReader(bytes.NewReader(contents))
	if err != nil {
		return fmt.Errorf("open package archive: %w", err)
	}
	defer gzipReader.Close()
	reader := tar.NewReader(gzipReader)
	var expanded int64
	files := 0
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read package archive: %w", err)
		}
		name := strings.TrimPrefix(path.Clean(strings.ReplaceAll(header.Name, "\\", "/")), "package/")
		if name == "." || name == "" {
			continue
		}
		if strings.HasPrefix(name, "/") || name == ".." || strings.HasPrefix(name, "../") {
			return fmt.Errorf("package archive contains unsafe path %q", header.Name)
		}
		target := filepath.Join(destination, filepath.FromSlash(name))
		if !withinDirectory(destination, target) {
			return fmt.Errorf("package archive escapes its destination")
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			files++
			expanded += header.Size
			if files > maxArchiveFiles || expanded > maxExpandedSize {
				return fmt.Errorf("package archive exceeds extraction limits")
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(0o644)
			if header.Mode&0o111 != 0 {
				mode = 0o755
			}
			file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return err
			}
			_, copyErr := io.CopyN(file, reader, header.Size)
			closeErr := file.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		default:
			return fmt.Errorf("package archive contains unsupported link or device %q", header.Name)
		}
	}
	if !fileExists(filepath.Join(destination, "package.json")) {
		return fmt.Errorf("package archive has no package.json")
	}
	return nil
}

func withinDirectory(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func equalBytes(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	var difference byte
	for index := range left {
		difference |= left[index] ^ right[index]
	}
	return difference == 0
}
