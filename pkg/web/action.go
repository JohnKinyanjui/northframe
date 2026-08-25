package web

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

// Action is a non-GET endpoint colocated with a page loader.
type Action struct {
	Method  string
	Path    string
	Handler ActionHandler
}

// ActionHandler handles a mutation through the same Context used by loaders.
type ActionHandler func(*Context) error

// FormActionHandler receives a decoded and validated Go input value. It keeps
// request parsing at the HTTP boundary while services own business rules.
type FormActionHandler[T any] func(*Context, T) (ActionResult, error)

func Post(path string, handler ActionHandler) Action {
	return Action{Method: http.MethodPost, Path: path, Handler: handler}
}

// PostForm creates a typed POST action. The input struct uses form, label, and
// validate tags; decoding failures become structured 422 action results.
func PostForm[T any](path string, handler FormActionHandler[T]) Action {
	inputType := reflect.TypeOf((*T)(nil)).Elem()
	if inputType.Kind() != reflect.Struct {
		panic(fmt.Sprintf("northframe: PostForm input %s must be a struct", inputType))
	}
	if handler == nil {
		panic("northframe: PostForm handler cannot be nil")
	}
	return Post(path, func(ctx *Context) error {
		input, fields, err := DecodeForm[T](ctx.Request)
		if err != nil {
			return BadRequest("could not read the submitted form", err)
		}
		if len(fields) > 0 {
			return writeActionResult(ctx, ActionInvalid("Please correct the highlighted fields.", fields))
		}
		result, err := handler(ctx, input)
		if err != nil {
			return err
		}
		return writeActionResult(ctx, result)
	})
}

func Put(path string, handler ActionHandler) Action {
	return Action{Method: http.MethodPut, Path: path, Handler: handler}
}

func Patch(path string, handler ActionHandler) Action {
	return Action{Method: http.MethodPatch, Path: path, Handler: handler}
}

func Delete(path string, handler ActionHandler) Action {
	return Action{Method: http.MethodDelete, Path: path, Handler: handler}
}

// HandleAction mounts a sidecar action relative to its page URL.
func (app *App) HandleAction(basePath string, action Action, middleware ...Middleware) {
	path := actionPath(basePath, action.Path)
	app.HandleFunc(action.Method+" "+path, func(writer http.ResponseWriter, request *http.Request) {
		if err := action.Handler(newContext(writer, request)); err != nil {
			app.HandleError(writer, request, err)
		}
	}, middleware...)
}

func actionPath(basePath, configured string) string {
	if configured == "" {
		return basePath
	}
	if strings.HasPrefix(configured, "/") {
		return configured
	}
	return strings.TrimRight(basePath, "/") + "/" + strings.TrimLeft(configured, "/")
}
