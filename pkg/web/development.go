package web

import (
	"net/http"
	"os"
	"strings"
)

const (
	developmentProbePath  = "/_northframe/dev"
	developmentSocketPath = "/_northframe/dev/reload"
)

func (app *App) enableDevelopmentReload() {
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("NORTHFRAME_ENV")), "development") {
		return
	}
	app.HandleFunc(http.MethodHead+" "+developmentProbePath, func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Cache-Control", "no-store")
		writer.WriteHeader(http.StatusNoContent)
	})
	app.WebSocket(developmentSocketPath, SocketOptions{
		ReadLimit:      1 << 10,
		MaxConnections: 10_000,
	}, func(_ *Context, socket *Socket) error {
		for {
			if _, _, err := socket.Read(socket.Context()); err != nil {
				return nil
			}
		}
	})
}
