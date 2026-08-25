package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBrowserHandlerServesModularRuntime(t *testing.T) {
	handler := BrowserHandler()
	for _, test := range []struct {
		path     string
		contains string
	}{
		{path: "/runtime.js", contains: `from "./reload.js"`},
		{path: "/reload.js", contains: "enableDevelopmentReload"},
		{path: "/component.js", contains: "mountComponent"},
		{path: "/forms.js", contains: "nf-enhance"},
		{path: "/calculator.js", contains: "mountCalculators"},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), test.contains) {
			t.Errorf("GET %s = (%d, %q)", test.path, response.Code, response.Body.String())
		}
		if contentType := response.Header().Get("Content-Type"); !strings.Contains(contentType, "javascript") {
			t.Errorf("GET %s Content-Type = %q", test.path, contentType)
		}
	}
}
