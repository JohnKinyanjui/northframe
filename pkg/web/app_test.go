package web

import (
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type testDependency struct{ Value string }

func TestAppInjectsTypedDependencies(t *testing.T) {
	app := New()
	Provide(app, &testDependency{Value: "ready"})
	app.HandleFunc("GET /", func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(MustUse[*testDependency](ContextFor(request)).Value))
	})

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Body.String() != "ready" {
		t.Fatalf("response = %q, want ready", response.Body.String())
	}
}

func TestAppHandlesRelativeActionsAndHTTPErrors(t *testing.T) {
	app := New()
	app.HandleAction("/users", Post("{id}/toggle", func(ctx *Context) error {
		if ctx.Param("id") != "42" {
			t.Fatalf("path id = %q", ctx.Param("id"))
		}
		return Error(http.StatusConflict, "cannot toggle", errors.New("locked"))
	}))

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/users/42/toggle", nil))
	if response.Code != http.StatusConflict || response.Body.String() != "cannot toggle\n" {
		t.Fatalf("response = (%d, %q)", response.Code, response.Body.String())
	}
}

func TestAppHandlesLoaderRedirectErrors(t *testing.T) {
	app := New()
	app.HandleFunc("GET /shipping", func(writer http.ResponseWriter, request *http.Request) {
		app.HandleError(writer, request, Redirect("/shipping/zones", http.StatusSeeOther))
	})

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/shipping", nil))
	if response.Code != http.StatusSeeOther {
		t.Fatalf("status = %d", response.Code)
	}
	if location := response.Header().Get("Location"); location != "/shipping/zones" {
		t.Fatalf("location = %q", location)
	}
}

func TestAppHandlesAbsoluteActionPaths(t *testing.T) {
	app := New()
	app.HandleAction("/account", Post("/signup", func(ctx *Context) error {
		return ctx.JSON(http.StatusCreated, map[string]bool{"created": true})
	}))
	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/signup", nil))
	if response.Code != http.StatusCreated || response.Body.String() != "{\"created\":true}\n" {
		t.Fatalf("response = (%d, %q)", response.Code, response.Body.String())
	}
}

func TestContextCarriesDatabaseServicesAndLocals(t *testing.T) {
	app := New()
	db := &sql.DB{}
	Provide(app, db)
	Provide(app, &testDependency{Value: "service"})
	app.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			SetLocal(request, "user", "Amina")
			next.ServeHTTP(writer, request)
		})
	})
	app.HandleFunc("GET /users/{id}", func(writer http.ResponseWriter, request *http.Request) {
		ctx := ContextFor(request)
		if ctx.DB != db || MustUse[*testDependency](ctx).Value != "service" || ctx.Locals["user"] != "Amina" {
			t.Fatalf("context was not populated: %#v", ctx)
		}
		if ctx.Param("id") != "7" || ctx.Query("tab") != "profile" {
			t.Fatalf("request helpers returned id=%q tab=%q", ctx.Param("id"), ctx.Query("tab"))
		}
		writer.WriteHeader(http.StatusNoContent)
	})
	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/users/7?tab=profile", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestAppRecoversPanics(t *testing.T) {
	app := New()
	app.HandleFunc("GET /", func(http.ResponseWriter, *http.Request) { panic("boom") })
	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", response.Code)
	}
}

func TestAppMountCoexistsWithRootPageAction(t *testing.T) {
	app := New()
	app.HandleFunc("POST /", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	})
	app.Mount("/api/v1/", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(request.Method + " " + request.URL.Path))
	}))

	apiResponse := httptest.NewRecorder()
	app.ServeHTTP(apiResponse, httptest.NewRequest(http.MethodPost, "/api/v1/products", nil))
	if apiResponse.Code != http.StatusOK || apiResponse.Body.String() != "POST /api/v1/products" {
		t.Fatalf("mounted response = (%d, %q)", apiResponse.Code, apiResponse.Body.String())
	}

	pageResponse := httptest.NewRecorder()
	app.ServeHTTP(pageResponse, httptest.NewRequest(http.MethodPost, "/", nil))
	if pageResponse.Code != http.StatusNoContent {
		t.Fatalf("page action status = %d, want %d", pageResponse.Code, http.StatusNoContent)
	}
}
