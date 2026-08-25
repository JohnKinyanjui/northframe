package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCSRFMiddlewareAcceptsMatchingFormToken(t *testing.T) {
	handler := CSRF()(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	initial := httptest.NewRecorder()
	handler.ServeHTTP(initial, httptest.NewRequest(http.MethodGet, "/", nil))
	cookie := initial.Result().Cookies()[0]

	form := url.Values{"_northframe_csrf": {cookie.Value}}
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", response.Code)
	}
}

func TestCSRFMiddlewareRejectsMissingToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "expected"})
	response := httptest.NewRecorder()
	CSRF()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
}

func TestCSRFTokenIsAvailableToServerRenderedForms(t *testing.T) {
	app := New()
	app.HandleFunc("GET /form", func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(CSRFToken(ContextFor(request))))
	}, CSRF())
	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/form", nil))
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || response.Body.String() != cookies[0].Value {
		t.Fatalf("rendered token = %q, cookies = %#v", response.Body.String(), cookies)
	}
}
