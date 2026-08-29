package auth

import (
	"errors"
	"net/http"

	"github.com/JohnKinyanjui/northframe/pkg/web"
)

const sessionLocal = "northframe.auth.session"

type GuardOptions struct {
	Permissions []string
	LoginPath   string
}

// Load adds a valid session to the Northframe request context when present.
// Anonymous requests continue normally.
func Load(manager *Manager) web.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			session, err := manager.Get(request.Context(), request)
			if err == nil {
				web.SetLocal(request, sessionLocal, session)
			} else if !errors.Is(err, ErrSessionNotFound) {
				http.Error(writer, "unable to load session", http.StatusInternalServerError)
				return
			}
			next.ServeHTTP(writer, request)
		})
	}
}

// Require rejects anonymous requests and sessions missing any required
// permission. LoginPath redirects anonymous browser requests when configured.
func Require(manager *Manager, options GuardOptions) web.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			session, err := manager.Get(request.Context(), request)
			if errors.Is(err, ErrSessionNotFound) {
				if options.LoginPath != "" {
					http.Redirect(writer, request, options.LoginPath, http.StatusSeeOther)
					return
				}
				http.Error(writer, "authentication required", http.StatusUnauthorized)
				return
			}
			if err != nil {
				http.Error(writer, "unable to load session", http.StatusInternalServerError)
				return
			}
			if !session.CanAll(options.Permissions...) {
				http.Error(writer, "permission denied", http.StatusForbidden)
				return
			}
			web.SetLocal(request, sessionLocal, session)
			next.ServeHTTP(writer, request)
		})
	}
}

// Current returns the session loaded by Load or Require.
func Current(ctx *web.Context) (Session, bool) {
	if ctx == nil {
		return Session{}, false
	}
	session, ok := ctx.Locals[sessionLocal].(Session)
	return cloneSession(session), ok
}

func MustCurrent(ctx *web.Context) Session {
	session, ok := Current(ctx)
	if !ok {
		panic("northframe: authenticated session is missing; add auth.Load or auth.Require middleware")
	}
	return session
}
