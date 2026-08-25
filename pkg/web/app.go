package web

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
)

// Middleware wraps a route handler. Middleware registered on App applies to
// every route; route sidecars can additionally expose route-local middleware.
type Middleware func(http.Handler) http.Handler

// App is Northframe's small net/http application container. It owns routing,
// dependency injection, middleware, and error responses without replacing the
// standard library HTTP stack.
type App struct {
	mux          *http.ServeMux
	dependencies map[reflect.Type]any
	middleware   []Middleware
	errorHandler func(http.ResponseWriter, *http.Request, error)
	mu           sync.RWMutex
}

// New creates an empty Northframe application.
func New() *App {
	app := &App{
		mux:          http.NewServeMux(),
		dependencies: make(map[reflect.Type]any),
	}
	app.errorHandler = app.defaultErrorHandler
	return app
}

// Use appends global middleware in declaration order.
func (app *App) Use(middleware ...Middleware) {
	app.middleware = append(app.middleware, middleware...)
}

// Handle registers a standard Go 1.22+ method-and-path pattern.
func (app *App) Handle(pattern string, handler http.Handler, middleware ...Middleware) {
	app.mux.Handle(pattern, chain(handler, middleware...))
}

// HandleFunc registers a handler function with optional route middleware.
func (app *App) HandleFunc(pattern string, handler http.HandlerFunc, middleware ...Middleware) {
	app.Handle(pattern, handler, middleware...)
}

// ServeHTTP injects application dependencies and runs global middleware.
func (app *App) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err := fmt.Errorf("panic: %v", recovered)
			if request.URL.Path == "/api" || strings.HasPrefix(request.URL.Path, "/api/") {
				app.handleAPIError(writer, err)
				return
			}
			app.HandleError(writer, request, err)
		}
	}()
	state := &requestState{app: app, locals: make(map[string]any)}
	request = request.WithContext(context.WithValue(request.Context(), requestStateContextKey{}, state))
	chain(app.mux, app.middleware...).ServeHTTP(writer, request)
}

// HandleError writes an error through the configured application handler.
func (app *App) HandleError(writer http.ResponseWriter, request *http.Request, err error) {
	app.errorHandler(writer, request, err)
}

// SetErrorHandler replaces the default plain-text error response.
func (app *App) SetErrorHandler(handler func(http.ResponseWriter, *http.Request, error)) {
	if handler != nil {
		app.errorHandler = handler
	}
}

// Provide registers a typed application dependency.
func Provide[T any](app *App, dependency T) {
	typeKey := reflect.TypeOf((*T)(nil)).Elem()
	app.mu.Lock()
	defer app.mu.Unlock()
	app.dependencies[typeKey] = dependency
}

// Resolve returns a typed dependency from the request's Northframe app.
func Resolve[T any](request *http.Request) (T, bool) {
	return Use[T](ContextFor(request))
}

// MustResolve returns a typed dependency or panics with an actionable message.
func MustResolve[T any](request *http.Request) T {
	dependency, ok := Resolve[T](request)
	if !ok {
		panic(missingDependencyMessage[T]())
	}
	return dependency
}

func chain(handler http.Handler, middleware ...Middleware) http.Handler {
	for index := len(middleware) - 1; index >= 0; index-- {
		handler = middleware[index](handler)
	}
	return handler
}

func dependencyFromApp[T any](app *App) (T, bool) {
	var zero T
	typeKey := reflect.TypeOf((*T)(nil)).Elem()
	app.mu.RLock()
	dependency, exists := app.dependencies[typeKey]
	app.mu.RUnlock()
	if !exists {
		return zero, false
	}
	result, ok := dependency.(T)
	return result, ok
}

func missingDependencyMessage[T any]() string {
	typeKey := reflect.TypeOf((*T)(nil)).Elem()
	return fmt.Sprintf("northframe: dependency %s was not provided", typeKey)
}
