package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPublicHandlerServesEmbeddedAssetAndETag(t *testing.T) {
	asset := NewAsset([]byte("font-data"), "font/woff2")
	handler := http.StripPrefix("/public/", PublicHandler(map[string]Asset{"fonts/app.woff2": asset}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/public/fonts/app.woff2", nil))
	if response.Code != http.StatusOK || response.Body.String() != "font-data" {
		t.Fatalf("response = (%d, %q)", response.Code, response.Body.String())
	}

	cached := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/public/fonts/app.woff2", nil)
	request.Header.Set("If-None-Match", asset.ETag)
	handler.ServeHTTP(cached, request)
	if cached.Code != http.StatusNotModified {
		t.Fatalf("cached status = %d, want 304", cached.Code)
	}
}
