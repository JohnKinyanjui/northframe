package web

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"path"
	"strings"
)

// Asset is one public file embedded into the generated Go router.
type Asset struct {
	Content     []byte
	ContentType string
	ETag        string
}

// NewAsset prepares immutable cache metadata for generated public content.
func NewAsset(content []byte, contentType string) Asset {
	digest := sha256.Sum256(content)
	return Asset{Content: content, ContentType: contentType, ETag: fmt.Sprintf(`"%x"`, digest[:12])}
}

// PublicHandler serves an exact map of compiler-embedded application assets.
func PublicHandler(assets map[string]Asset) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		name := strings.TrimPrefix(path.Clean("/"+request.URL.Path), "/")
		asset, ok := assets[name]
		if !ok {
			http.NotFound(writer, request)
			return
		}
		if request.Header.Get("If-None-Match") == asset.ETag {
			writer.WriteHeader(http.StatusNotModified)
			return
		}
		writer.Header().Set("Content-Type", asset.ContentType)
		writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		writer.Header().Set("ETag", asset.ETag)
		_, _ = writer.Write(asset.Content)
	})
}
