package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const maxJSONBody = 1 << 20

// Context is the framework request passed to page/layout loaders and actions.
// It keeps net/http available while exposing common application facilities.
type Context struct {
	Request  *http.Request
	Response http.ResponseWriter
	DB       *sql.DB
	Locals   map[string]any

	app *App
}

type requestState struct {
	app    *App
	locals map[string]any
}

// ContextFor returns the Northframe context associated with a request.
func ContextFor(request *http.Request) *Context {
	return newContext(nil, request)
}

func newContext(writer http.ResponseWriter, request *http.Request) *Context {
	state, _ := request.Context().Value(requestStateContextKey{}).(*requestState)
	current := &Context{Request: request, Response: writer, Locals: make(map[string]any)}
	if state == nil {
		return current
	}
	current.app = state.app
	current.Locals = state.locals
	current.DB, _ = dependencyFromApp[*sql.DB](state.app)
	return current
}

// StdContext returns the cancellation-aware standard Go request context.
func (current *Context) StdContext() context.Context {
	return current.Request.Context()
}

// DetachedContext retains request values but is not cancelled when the HTTP
// request finishes. It is useful when handing work to a queue or scheduler.
func (current *Context) DetachedContext() context.Context {
	return context.WithoutCancel(current.Request.Context())
}

func (current *Context) Param(name string) string {
	return current.Request.PathValue(name)
}

func (current *Context) Query(name string) string {
	return current.Request.URL.Query().Get(name)
}

func (current *Context) Path() string {
	return current.Request.URL.Path
}

func (current *Context) Form() (url.Values, error) {
	if err := current.Request.ParseForm(); err != nil {
		return nil, err
	}
	return current.Request.Form, nil
}

func (current *Context) FormValue(name string) string {
	return current.Request.FormValue(name)
}

func (current *Context) Cookie(name string) (*http.Cookie, error) {
	return current.Request.Cookie(name)
}

func (current *Context) SetCookie(cookie *http.Cookie) error {
	if current.Response == nil {
		return errors.New("cannot set a cookie from a page or layout loader")
	}
	http.SetCookie(current.Response, cookie)
	return nil
}

func (current *Context) Redirect(path string, status int) error {
	if current.Response == nil {
		return errors.New("cannot redirect from a page or layout loader")
	}
	http.Redirect(current.Response, current.Request, path, status)
	return nil
}

func (current *Context) NoContent() error {
	if current.Response == nil {
		return errors.New("cannot write a response from a page or layout loader")
	}
	current.Response.WriteHeader(http.StatusNoContent)
	return nil
}

func (current *Context) JSON(status int, value any) error {
	if current.Response == nil {
		return errors.New("cannot write a response from a page or layout loader")
	}
	current.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
	current.Response.WriteHeader(status)
	return json.NewEncoder(current.Response).Encode(value)
}

// DecodeJSON reads one strict JSON value with a 1 MiB body limit. Unknown
// object fields are rejected so API request contracts do not silently drift.
func (current *Context) DecodeJSON(value any) error {
	if current.Request == nil {
		return errors.New("cannot decode JSON without a request")
	}
	body := current.Request.Body
	if current.Response != nil {
		body = http.MaxBytesReader(current.Response, body, maxJSONBody)
	} else {
		body = io.NopCloser(io.LimitReader(body, maxJSONBody+1))
	}
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return errors.New("decode JSON: request body must contain one value")
		}
		return fmt.Errorf("decode JSON: %w", err)
	}
	return nil
}

// Use returns a typed application service from the request context. Database
// query sets, mailers, queues, caches, and schedulers can all use this path.
func Use[T any](current *Context) (T, bool) {
	if current == nil || current.app == nil {
		var zero T
		return zero, false
	}
	return dependencyFromApp[T](current.app)
}

// MustUse returns a typed application service or panics with an actionable message.
func MustUse[T any](current *Context) T {
	dependency, ok := Use[T](current)
	if !ok {
		panic(missingDependencyMessage[T]())
	}
	return dependency
}

// SetLocal exposes request-scoped data created in net/http middleware to
// loaders and actions through Context.Locals.
func SetLocal(request *http.Request, name string, value any) {
	if state, ok := request.Context().Value(requestStateContextKey{}).(*requestState); ok {
		state.locals[name] = value
	}
}

type requestStateContextKey struct{}
