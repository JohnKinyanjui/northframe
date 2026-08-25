package web

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
)

const csrfCookieName = "northframe_csrf"

// CSRF protects unsafe requests with a signed-random double-submit token. The
// embedded browser runtime adds the matching hidden field to non-GET forms.
func CSRF() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			cookie, err := request.Cookie(csrfCookieName)
			if err != nil || cookie.Value == "" {
				cookie = &http.Cookie{
					Name: csrfCookieName, Value: newCSRFToken(), Path: "/",
					Secure: request.TLS != nil, HttpOnly: false, SameSite: http.SameSiteStrictMode,
				}
				http.SetCookie(writer, cookie)
			}
			if isSafeMethod(request.Method) {
				next.ServeHTTP(writer, request)
				return
			}
			token := request.Header.Get("X-CSRF-Token")
			if token == "" {
				if err := request.ParseForm(); err == nil {
					token = request.Form.Get("_northframe_csrf")
				}
			}
			if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(token)) != 1 {
				http.Error(writer, "invalid CSRF token", http.StatusForbidden)
				return
			}
			next.ServeHTTP(writer, request)
		})
	}
}

func newCSRFToken() string {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		panic("northframe: cannot generate CSRF token: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(buffer)
}

func isSafeMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions || method == http.MethodTrace
}
