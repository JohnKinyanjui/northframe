package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed browser/*.js
var browserAssets embed.FS

// BrowserHandler serves the tiny hydration runtime embedded in the Go binary.
func BrowserHandler() http.Handler {
	assets, err := fs.Sub(browserAssets, "browser")
	if err != nil {
		panic("northframe: embedded browser assets are unavailable: " + err.Error())
	}
	return http.FileServer(http.FS(assets))
}
