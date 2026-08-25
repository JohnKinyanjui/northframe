package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDevelopmentReloadEndpointsAreEnvironmentScoped(t *testing.T) {
	t.Run("production", func(t *testing.T) {
		t.Setenv("NORTHFRAME_ENV", "production")
		app := New()
		response := httptest.NewRecorder()
		app.ServeHTTP(response, httptest.NewRequest(http.MethodHead, developmentProbePath, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
		}
	})

	t.Run("development", func(t *testing.T) {
		t.Setenv("NORTHFRAME_ENV", "development")
		app := New()
		response := httptest.NewRecorder()
		app.ServeHTTP(response, httptest.NewRequest(http.MethodHead, developmentProbePath, nil))
		if response.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
		}
		if cacheControl := response.Header().Get("Cache-Control"); cacheControl != "no-store" {
			t.Fatalf("Cache-Control = %q, want no-store", cacheControl)
		}
	})
}
