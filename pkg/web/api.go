package web

import (
	"encoding/json"
	"log"
	"net/http"
)

// APIHandler is a convention-discovered handler from the application's api tree.
type APIHandler func(*Context) error

// HandleAPI registers a context-aware API handler with optional middleware.
func (app *App) HandleAPI(method, path string, handler APIHandler, middleware ...Middleware) {
	app.HandleFunc(method+" "+path, func(writer http.ResponseWriter, request *http.Request) {
		if err := handler(newContext(writer, request)); err != nil {
			app.handleAPIError(writer, err)
		}
	}, middleware...)
}

func (app *App) handleAPIError(writer http.ResponseWriter, err error) {
	status, message := publicError(err)
	if status >= http.StatusInternalServerError {
		log.Printf("northframe: API request failed: %v", err)
	}
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(map[string]any{
		"error": map[string]any{"status": status, "message": message},
	})
}
