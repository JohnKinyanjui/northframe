package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleAPIProvidesContextAndJSONErrors(t *testing.T) {
	app := New()
	app.HandleAPI(http.MethodGet, "/api/users/{id}", func(ctx *Context) error {
		if ctx.Param("id") != "42" || ctx.Response == nil {
			t.Fatalf("API context = %#v", ctx)
		}
		return Error(http.StatusConflict, "user is locked", errors.New("private detail"))
	})
	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/users/42", nil))
	if response.Code != http.StatusConflict || response.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("response = (%d, %q)", response.Code, response.Header().Get("Content-Type"))
	}
	if body := response.Body.String(); !strings.Contains(body, `"message":"user is locked"`) || strings.Contains(body, "private detail") {
		t.Fatalf("body = %s", body)
	}
}

func TestContextDecodeJSONIsStrict(t *testing.T) {
	type input struct {
		Name string `json:"name"`
	}
	request := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(`{"name":"Amina"}`))
	var value input
	if err := ContextFor(request).DecodeJSON(&value); err != nil || value.Name != "Amina" {
		t.Fatalf("DecodeJSON() = (%#v, %v)", value, err)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(`{"name":"Amina","role":"admin"}`))
	if err := ContextFor(request).DecodeJSON(&value); err == nil {
		t.Fatal("DecodeJSON accepted an unknown field")
	}
}

func TestAPIPanicUsesSafeJSONError(t *testing.T) {
	app := New()
	app.HandleAPI(http.MethodGet, "/api/panic", func(*Context) error { panic("database secret") })
	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/panic", nil))
	if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), `"message":"Internal Server Error"`) {
		t.Fatalf("response = (%d, %s)", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "database secret") {
		t.Fatalf("panic detail leaked: %s", response.Body.String())
	}
}
