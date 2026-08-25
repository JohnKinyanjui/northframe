package web

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCustomErrorRendererReceivesSafeRequestData(t *testing.T) {
	app := New()
	app.SetErrorRenderer(func(writer io.Writer, page ErrorPage) error {
		_, err := fmt.Fprintf(writer, "<h1>%d</h1><p>%s</p><code>%s</code><small>%s</small>", page.Status, page.Message, page.Path, page.RequestID)
		return err
	})
	app.HandleFunc("GET /private", func(writer http.ResponseWriter, request *http.Request) {
		app.HandleError(writer, request, Error(http.StatusTeapot, "Tea is unavailable", errors.New("private database detail")))
	})

	request := httptest.NewRequest(http.MethodGet, "/private", nil)
	request.Header.Set("X-Request-ID", "req-42")
	response := httptest.NewRecorder()
	app.ServeHTTP(response, request)
	if response.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusTeapot)
	}
	if contentType := response.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Fatalf("Content-Type = %q", contentType)
	}
	body := response.Body.String()
	for _, expected := range []string{"418", "Tea is unavailable", "/private", "req-42"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("body %q does not contain %q", body, expected)
		}
	}
	if strings.Contains(body, "private database detail") {
		t.Fatalf("custom error page leaked the wrapped cause: %q", body)
	}
}

func TestCustomErrorRendererPreservesLoaderRedirect(t *testing.T) {
	app := New()
	app.SetErrorRenderer(func(io.Writer, ErrorPage) error {
		t.Fatal("renderer was called for a redirect")
		return nil
	})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/old", nil)
	app.HandleError(response, request, Redirect("/new", http.StatusTemporaryRedirect))
	if response.Code != http.StatusTemporaryRedirect || response.Header().Get("Location") != "/new" {
		t.Fatalf("redirect = (%d, %q)", response.Code, response.Header().Get("Location"))
	}
}

func TestCustomErrorRendererFallsBackWhenRenderingFails(t *testing.T) {
	app := New()
	app.SetErrorRenderer(func(io.Writer, ErrorPage) error { return errors.New("template failed") })
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/broken", nil)
	app.HandleError(response, request, NotFound("Page missing"))
	if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), "Page missing") {
		t.Fatalf("fallback = (%d, %q)", response.Code, response.Body.String())
	}
}
