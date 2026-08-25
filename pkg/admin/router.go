package admin

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"northframe.dev/northframe/pkg/auth"
	"northframe.dev/northframe/pkg/web"
)

type Options struct {
	BasePath  string
	LoginPath string
	Title     string
	PageSize  int
}

type adminServer struct {
	app      *web.App
	registry *Registry
	options  Options
	renderer *renderer
	session  SessionResolver
}

// SessionResolver maps application authentication state into the permission
// session consumed by the admin registry. Existing applications can keep their
// own cookies, JWTs, database models, and authentication middleware.
type SessionResolver func(*web.Context) (auth.Session, bool)

// Access describes how an existing application protects admin routes and how
// the authenticated request is mapped into a Northframe permission session.
type Access struct {
	Middleware []web.Middleware
	Session    SessionResolver
}

// Mount registers the framework-owned administration dashboard and CRUD
// routes. Repositories remain application services; admin handlers never issue
// SQL directly.
func Mount(app *web.App, registry *Registry, sessions *auth.Manager, options Options) error {
	if app == nil || registry == nil || sessions == nil {
		return errors.New("admin app, registry, and session manager are required")
	}
	options = normalizeOptions(options)
	return MountWithAccess(app, registry, Access{
		Middleware: []web.Middleware{auth.Require(sessions, auth.GuardOptions{LoginPath: options.LoginPath})},
		Session:    auth.Current,
	}, options)
}

// MountWithAccess registers the administration UI using application-owned
// authentication. The supplied middleware must reject anonymous requests and
// the resolver must return the authenticated subject and admin permissions.
func MountWithAccess(app *web.App, registry *Registry, access Access, options Options) error {
	if app == nil || registry == nil {
		return errors.New("admin app and registry are required")
	}
	if len(access.Middleware) == 0 {
		return errors.New("admin access middleware is required")
	}
	if access.Session == nil {
		return errors.New("admin session resolver is required")
	}
	options = normalizeOptions(options)
	renderer, err := newRenderer()
	if err != nil {
		return err
	}
	server := &adminServer{app: app, registry: registry, options: options, renderer: renderer, session: access.Session}
	protected := append([]web.Middleware(nil), access.Middleware...)
	protected = append(protected, web.CSRF())

	app.HandleFunc("GET "+options.BasePath+"/{$}", server.wrap(server.dashboard), protected...)
	app.HandleFunc("GET "+options.BasePath+"/{resource}", server.wrap(server.list), protected...)
	app.HandleFunc("GET "+options.BasePath+"/{resource}/new", server.wrap(server.newRecord), protected...)
	app.HandleFunc("POST "+options.BasePath+"/{resource}", server.wrap(server.create), protected...)
	app.HandleFunc("GET "+options.BasePath+"/{resource}/{id}", server.wrap(server.edit), protected...)
	app.HandleFunc("POST "+options.BasePath+"/{resource}/{id}", server.wrap(server.update), protected...)
	app.HandleFunc("POST "+options.BasePath+"/{resource}/{id}/delete", server.wrap(server.delete), protected...)
	return nil
}

type adminHandler func(http.ResponseWriter, *http.Request, auth.Session) error

func (server *adminServer) wrap(handler adminHandler) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		session, ok := server.session(web.ContextFor(request))
		if !ok || strings.TrimSpace(session.Subject) == "" {
			server.app.HandleError(writer, request, web.Unauthorized("authentication required"))
			return
		}
		request = request.WithContext(auth.WithSession(request.Context(), session))
		if err := handler(writer, request, session); err != nil {
			server.app.HandleError(writer, request, err)
		}
	}
}

func (server *adminServer) dashboard(writer http.ResponseWriter, request *http.Request, session auth.Session) error {
	return server.renderer.render(writer, dashboardView{
		Title: server.options.Title, BasePath: server.options.BasePath,
		Resources: server.registry.VisibleResources(session),
		CSRFToken: web.CSRFToken(web.ContextFor(request)),
	})
}

func (server *adminServer) list(writer http.ResponseWriter, request *http.Request, session auth.Session) error {
	resource, err := server.allowedResource(request, session, permissionView)
	if err != nil {
		return err
	}
	pageNumber := positiveInt(request.URL.Query().Get("page"), 1)
	page, err := resource.Repository.List(request.Context(), ListQuery{
		Search: request.URL.Query().Get("q"), Sort: request.URL.Query().Get("sort"),
		Desc: request.URL.Query().Get("direction") == "desc", Page: pageNumber, PageSize: server.options.PageSize,
	})
	if err != nil {
		return fmt.Errorf("list admin resource %s: %w", resource.Name, err)
	}
	return server.renderer.render(writer, resourceListView(server, request, session, resource, page, pageNumber))
}

func (server *adminServer) newRecord(writer http.ResponseWriter, request *http.Request, session auth.Session) error {
	resource, err := server.allowedResource(request, session, permissionCreate)
	if err != nil {
		return err
	}
	return server.renderer.render(writer, resourceFormView(server, request, session, resource, Record{}, nil, true))
}

func (server *adminServer) create(writer http.ResponseWriter, request *http.Request, session auth.Session) error {
	resource, err := server.allowedResource(request, session, permissionCreate)
	if err != nil {
		return err
	}
	record, fields, err := decodeRecord(request, resource)
	if err != nil {
		return web.BadRequest("could not read the admin form", err)
	}
	if len(fields) > 0 {
		writer.WriteHeader(http.StatusUnprocessableEntity)
		return server.renderer.render(writer, resourceFormView(server, request, session, resource, record, fields, true))
	}
	if _, err := resource.Repository.Create(request.Context(), record); err != nil {
		return fmt.Errorf("create admin resource %s: %w", resource.Name, err)
	}
	http.Redirect(writer, request, server.options.BasePath+"/"+resource.Name, http.StatusSeeOther)
	return nil
}

func (server *adminServer) edit(writer http.ResponseWriter, request *http.Request, session auth.Session) error {
	resource, err := server.allowedResource(request, session, permissionView)
	if err != nil {
		return err
	}
	record, err := resource.Repository.Get(request.Context(), request.PathValue("id"))
	if err != nil {
		return fmt.Errorf("get admin resource %s: %w", resource.Name, err)
	}
	return server.renderer.render(writer, resourceFormView(server, request, session, resource, record, nil, false))
}

func (server *adminServer) update(writer http.ResponseWriter, request *http.Request, session auth.Session) error {
	resource, err := server.allowedResource(request, session, permissionUpdate)
	if err != nil {
		return err
	}
	record, fields, err := decodeRecord(request, resource)
	if err != nil {
		return web.BadRequest("could not read the admin form", err)
	}
	if len(fields) > 0 {
		writer.WriteHeader(http.StatusUnprocessableEntity)
		return server.renderer.render(writer, resourceFormView(server, request, session, resource, record, fields, false))
	}
	id := request.PathValue("id")
	if _, err := resource.Repository.Update(request.Context(), id, record); err != nil {
		return fmt.Errorf("update admin resource %s: %w", resource.Name, err)
	}
	http.Redirect(writer, request, server.options.BasePath+"/"+resource.Name+"/"+id, http.StatusSeeOther)
	return nil
}

func (server *adminServer) delete(writer http.ResponseWriter, request *http.Request, session auth.Session) error {
	resource, err := server.allowedResource(request, session, permissionDelete)
	if err != nil {
		return err
	}
	if err := resource.Repository.Delete(request.Context(), request.PathValue("id")); err != nil {
		return fmt.Errorf("delete admin resource %s: %w", resource.Name, err)
	}
	http.Redirect(writer, request, server.options.BasePath+"/"+resource.Name, http.StatusSeeOther)
	return nil
}

type permissionOperation int

const (
	permissionView permissionOperation = iota
	permissionCreate
	permissionUpdate
	permissionDelete
)

func (server *adminServer) allowedResource(request *http.Request, session auth.Session, operation permissionOperation) (Resource, error) {
	resource, ok := server.registry.Resource(request.PathValue("resource"))
	if !ok {
		return Resource{}, web.NotFound("admin resource not found")
	}
	permission := resource.Permissions.View
	switch operation {
	case permissionCreate:
		permission = resource.Permissions.Create
	case permissionUpdate:
		permission = resource.Permissions.Update
	case permissionDelete:
		permission = resource.Permissions.Delete
	}
	if !session.Can(permission) {
		return Resource{}, web.Forbidden("permission denied")
	}
	return resource, nil
}

func normalizeOptions(options Options) Options {
	options.BasePath = "/" + strings.Trim(strings.TrimSpace(options.BasePath), "/")
	if options.BasePath == "/" {
		options.BasePath = "/admin"
	}
	if options.Title == "" {
		options.Title = "Administration"
	}
	if options.PageSize <= 0 || options.PageSize > 200 {
		options.PageSize = 50
	}
	return options
}

func positiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
