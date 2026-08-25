package web

import "net/http"

// mountedHTTPMethods contains the standard methods accepted by Mount. Using
// method-qualified ServeMux patterns keeps a mounted subtree compatible with
// Northframe page actions such as "POST /".
var mountedHTTPMethods = [...]string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodConnect,
	http.MethodOptions,
	http.MethodTrace,
}

// Mount registers an existing http.Handler at a path or subtree for every
// standard HTTP method. Use it for an Echo, Chi, or net/http application that
// must share Northframe's server without a reverse proxy.
func (app *App) Mount(pattern string, handler http.Handler, middleware ...Middleware) {
	for _, method := range mountedHTTPMethods {
		app.Handle(method+" "+pattern, handler, middleware...)
	}
}
