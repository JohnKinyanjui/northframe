package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildRoutesCreatesTypedNestedRouter(t *testing.T) {
	appRoot := t.TempDir()
	root := filepath.Join(appRoot, "routes")
	writeRouteTestFile(t, filepath.Join(root, "layout.north"), `<!doctype html><html><head></head><body><slot /></body></html>`)
	writeRouteTestFile(t, filepath.Join(root, "page.north"), `<h1 class="text-4xl">{Props.Title}</h1>`)
	writeRouteTestFile(t, filepath.Join(root, "layout.north.go"), `package routes
import "github.com/JohnKinyanjui/northframe/pkg/web"
type LayoutProps struct{}
func Layout(*web.Context) (LayoutProps, error) { return LayoutProps{}, nil }
`)
	writeRouteTestFile(t, filepath.Join(root, "page.north.go"), `package routes
import "github.com/JohnKinyanjui/northframe/pkg/web"
type PageProps struct { Title string }
func Page(*web.Context) (PageProps, error) { return PageProps{Title: "Home"}, nil }
`)
	writeRouteTestFile(t, filepath.Join(root, "auth", "layout.north"), `<section class="p-6"><slot /></section>`)
	writeRouteTestFile(t, filepath.Join(root, "auth", "page.north"), `<p>Sign in</p>`)
	writeRouteTestFile(t, filepath.Join(root, "auth", "layout.north.go"), `package auth
import "github.com/JohnKinyanjui/northframe/pkg/web"
type LayoutProps struct{}
func Layout(*web.Context) (LayoutProps, error) { return LayoutProps{}, nil }
`)
	writeRouteTestFile(t, filepath.Join(root, "auth", "page.north.go"), `package auth
import (
  "github.com/JohnKinyanjui/northframe/pkg/web"
)
type PageProps struct{}
func Page(*web.Context) (PageProps, error) { return PageProps{}, nil }
func PageActions() []web.Action { return nil }
`)
	writeRouteTestFile(t, filepath.Join(root, "auth", "page.css"), `.auth { color: rebeccapurple; }`)
	writeRouteTestFile(t, filepath.Join(appRoot, "app.css"), `:root { --brand: #252a31; } .global-shell { color: var(--brand); }`)
	writeRouteTestFile(t, filepath.Join(appRoot, "public", "fonts", "admin.woff2"), `font-data`)
	writeRouteTestFile(t, filepath.Join(appRoot, "public", "site.webmanifest"), `{}`)
	writeRouteTestFile(t, filepath.Join(appRoot, "components", "notice.north"), `<aside class="p-4">Shared notice</aside>`)
	writeRouteTestFile(t, filepath.Join(root, "auth", "page.north"), `<p>Sign in</p>{#include notice}`)

	build, err := BuildRoutes(root, "routesgen", "example.test/app/routes")
	if err != nil {
		t.Fatalf("BuildRoutes returned an error: %v", err)
	}
	if build.RouteCount != 2 {
		t.Fatalf("RouteCount = %d, want 2", build.RouteCount)
	}
	router := string(build.Files["router_generated.go"])
	generated := router + string(build.Files["auth_page_generated.go"])
	for _, expected := range []string{
		`func Register(app *web.App)`,
		`app.HandleFunc("GET /{$}"`,
		`app.HandleFunc("GET /auth"`,
		`routeAuth.Page(context)`,
		`routeRoot.Layout(context)`,
		`routeAuth.PageActions()`,
		`GET /public/`,
		`GET /fonts/admin.woff2`,
		`GET /site.webmanifest`,
		`data-north-route-kind=\"page\"`,
		`data-north-route-kind=\"layout\" data-north-route-segment=\"auth\"`,
		`fonts/admin.woff2`,
		`application/manifest+json`,
		`Shared notice`,
		`.text-4xl{font-size:2.25rem;line-height:2.5rem}`,
		`.auth { color: rebeccapurple; }`,
		`.global-shell { color: var(--brand); }`,
	} {
		if !strings.Contains(generated, expected) {
			t.Errorf("generated router does not contain %q\n%s", expected, router)
		}
	}
}

func TestBuildRoutesForwardsLayoutSlotThroughComponent(t *testing.T) {
	appRoot := t.TempDir()
	root := filepath.Join(appRoot, "routes")
	writeRouteTestFile(t, filepath.Join(root, "layout.north"), `<!doctype html><html><head></head><body><Shell><slot /></Shell></body></html>`)
	writeRouteTestFile(t, filepath.Join(root, "layout.north.go"), loaderSidecar("routes", "Layout"))
	writeRouteTestFile(t, filepath.Join(root, "page.north"), `<p>Forwarded page</p>`)
	writeRouteTestFile(t, filepath.Join(root, "page.north.go"), loaderSidecar("routes", "Page"))
	writeRouteTestFile(t, filepath.Join(appRoot, "components", "shell.north"), `<main><slot /></main>`)

	build, err := BuildRoutes(root, "routesgen", "example.test/app/routes")
	if err != nil {
		t.Fatal(err)
	}
	layout := string(build.Files["root_layout_generated.go"])
	if !strings.Contains(layout, `if content != nil`) {
		t.Fatalf("layout component did not forward the route slot\n%s", layout)
	}
}

func TestBuildRoutesCompilesRootErrorPageAndRegistersFallback(t *testing.T) {
	appRoot := t.TempDir()
	root := filepath.Join(appRoot, "routes")
	writeRouteTestFile(t, filepath.Join(root, "layout.north"), `<!doctype html><html><head></head><body><slot /></body></html>`)
	writeRouteTestFile(t, filepath.Join(root, "layout.north.go"), loaderSidecar("routes", "Layout"))
	writeRouteTestFile(t, filepath.Join(root, "page.north"), `<p>Home</p>`)
	writeRouteTestFile(t, filepath.Join(root, "page.north.go"), loaderSidecar("routes", "Page"))
	writeRouteTestFile(t, filepath.Join(root, "error.north"), `<!doctype html><html><head><title>${Props.Status}</title></head><body><h1>${Props.Message}</h1><p>${Props.Path}</p></body></html>`)

	build, err := BuildRoutes(root, "routesgen", "example.test/app/routes")
	if err != nil {
		t.Fatal(err)
	}
	errorRenderer := string(build.Files["application_error_generated.go"])
	router := string(build.Files["router_generated.go"])
	for _, expected := range []string{`type ApplicationErrorProps struct`, `web.WriteEscaped(w, props.Status)`, `web.WriteEscaped(w, props.Message)`} {
		if !strings.Contains(errorRenderer, expected) {
			t.Errorf("generated error renderer does not contain %q\n%s", expected, errorRenderer)
		}
	}
	for _, expected := range []string{`app.SetErrorRenderer`, `RenderApplicationError`, `app.HandleFunc("GET /"`, `web.NotFound("Page not found")`} {
		if !strings.Contains(router, expected) {
			t.Errorf("generated router does not contain %q\n%s", expected, router)
		}
	}
}

func TestBuildRoutesRejectsNestedErrorPageUntilBoundariesAreSupported(t *testing.T) {
	root := t.TempDir()
	writeRouteTestFile(t, filepath.Join(root, "layout.north"), `<!doctype html><html><head></head><body><slot /></body></html>`)
	writeRouteTestFile(t, filepath.Join(root, "layout.north.go"), loaderSidecar("routes", "Layout"))
	writeRouteTestFile(t, filepath.Join(root, "page.north"), `<p>Home</p>`)
	writeRouteTestFile(t, filepath.Join(root, "page.north.go"), loaderSidecar("routes", "Page"))
	writeRouteTestFile(t, filepath.Join(root, "admin", "error.north"), `<!doctype html><html><head></head><body>Error</body></html>`)

	_, err := BuildRoutes(root, "routesgen", "example.test/app/routes")
	if err == nil || !strings.Contains(err.Error(), "error.north currently belongs at the root") {
		t.Fatalf("BuildRoutes error = %v", err)
	}
}

func TestBuildRoutesCreatesDynamicAndCatchAllPatterns(t *testing.T) {
	root := t.TempDir()
	writeRouteTestFile(t, filepath.Join(root, "layout.north"), `<!doctype html><html><head></head><body><slot /></body></html>`)
	writeRouteTestFile(t, filepath.Join(root, "layout.north.go"), loaderSidecar("routes", "Layout"))
	writeRouteTestFile(t, filepath.Join(root, "page.north"), `<p>Home</p>`)
	writeRouteTestFile(t, filepath.Join(root, "page.north.go"), loaderSidecar("routes", "Page"))
	writeRouteTestFile(t, filepath.Join(root, "users", "id_", "page.north"), `<p>{Props.ID}</p>`)
	writeRouteTestFile(t, filepath.Join(root, "users", "id_", "page.north.go"), `package id
import "github.com/JohnKinyanjui/northframe/pkg/web"
type PageProps struct { ID string }
func Page(ctx *web.Context) (PageProps, error) { return PageProps{ID: ctx.Param("id")}, nil }
`)
	writeRouteTestFile(t, filepath.Join(root, "files", "path__", "page.north"), `<p>{Props.Path}</p>`)
	writeRouteTestFile(t, filepath.Join(root, "files", "path__", "page.north.go"), `package path
import "github.com/JohnKinyanjui/northframe/pkg/web"
type PageProps struct { Path string }
func Page(ctx *web.Context) (PageProps, error) { return PageProps{Path: ctx.Param("path")}, nil }
`)

	build, err := BuildRoutes(root, "routesgen", "example.test/app/routes")
	if err != nil {
		t.Fatal(err)
	}
	router := string(build.Files["router_generated.go"])
	for _, pattern := range []string{`GET /users/{id}`, `GET /files/{path...}`} {
		if !strings.Contains(router, pattern) {
			t.Errorf("router does not contain %q\n%s", pattern, router)
		}
	}
}

func TestBuildProjectDiscoversAPIMethodHandlers(t *testing.T) {
	appRoot := t.TempDir()
	routes := filepath.Join(appRoot, "routes")
	api := filepath.Join(routes, "api")
	writeRouteTestFile(t, filepath.Join(routes, "layout.north"), `<!doctype html><html><head></head><body><slot /></body></html>`)
	writeRouteTestFile(t, filepath.Join(routes, "layout.north.go"), loaderSidecar("routes", "Layout"))
	writeRouteTestFile(t, filepath.Join(routes, "page.north"), `<p>Home</p>`)
	writeRouteTestFile(t, filepath.Join(routes, "page.north.go"), loaderSidecar("routes", "Page"))
	writeRouteTestFile(t, filepath.Join(api, "products", "get.go"), `package products
import "github.com/JohnKinyanjui/northframe/pkg/web"
func GET(ctx *web.Context) error { return ctx.JSON(200, nil) }
func Middleware() []web.Middleware { return nil }
`)
	writeRouteTestFile(t, filepath.Join(api, "products", "post.go"), `package products
import "github.com/JohnKinyanjui/northframe/pkg/web"
func POST(ctx *web.Context) error { return ctx.JSON(201, nil) }
`)
	writeRouteTestFile(t, filepath.Join(api, "products", "id_", "delete.go"), `package product
import "github.com/JohnKinyanjui/northframe/pkg/web"
func DELETE(ctx *web.Context) error { return ctx.NoContent() }
`)

	build, err := BuildProject(routes, api, "routesgen", "example.test/app/routes", "example.test/app/routes/api")
	if err != nil {
		t.Fatal(err)
	}
	if build.RouteCount != 4 {
		t.Fatalf("RouteCount = %d, want one page and three API methods", build.RouteCount)
	}
	router := string(build.Files["router_generated.go"])
	for _, expected := range []string{
		`app.HandleAPI("GET", "/api/products", apiProducts.GET, apiProductsMiddleware...)`,
		`app.HandleAPI("POST", "/api/products", apiProducts.POST, apiProductsMiddleware...)`,
		`app.HandleAPI("DELETE", "/api/products/{id}", apiProductsId.DELETE)`,
	} {
		if !strings.Contains(router, expected) {
			t.Errorf("generated API router does not contain %q\n%s", expected, router)
		}
	}
}

func TestBuildProjectDiscoversAPIWebSocketHandler(t *testing.T) {
	appRoot := t.TempDir()
	routes := filepath.Join(appRoot, "routes")
	api := filepath.Join(routes, "api")
	writeRouteTestFile(t, filepath.Join(routes, "layout.north"), `<!doctype html><html><head></head><body><slot /></body></html>`)
	writeRouteTestFile(t, filepath.Join(routes, "layout.north.go"), loaderSidecar("routes", "Layout"))
	writeRouteTestFile(t, filepath.Join(routes, "page.north"), `<p>Home</p>`)
	writeRouteTestFile(t, filepath.Join(routes, "page.north.go"), loaderSidecar("routes", "Page"))
	writeRouteTestFile(t, filepath.Join(api, "events", "route.go"), `package events
import "github.com/JohnKinyanjui/northframe/pkg/web"
func WEBSOCKET(ctx *web.Context, socket *web.Socket) error { return nil }
func WebSocketOptions() web.SocketOptions { return web.SocketOptions{ReadLimit: 4096} }
func Middleware() []web.Middleware { return nil }
`)

	build, err := BuildProject(routes, api, "routesgen", "example.test/app/routes", "example.test/app/routes/api")
	if err != nil {
		t.Fatal(err)
	}
	if build.RouteCount != 2 {
		t.Fatalf("RouteCount = %d, want one page and one WebSocket", build.RouteCount)
	}
	router := string(build.Files["router_generated.go"])
	expected := `app.WebSocket("/api/events", apiEvents.WebSocketOptions(), apiEvents.WEBSOCKET, apiEventsMiddleware...)`
	if !strings.Contains(router, expected) {
		t.Fatalf("generated API router does not contain %q\n%s", expected, router)
	}
}

func TestBuildProjectRejectsConflictingGETAndWebSocket(t *testing.T) {
	appRoot := t.TempDir()
	routes := filepath.Join(appRoot, "routes")
	api := filepath.Join(routes, "api")
	writeRouteTestFile(t, filepath.Join(routes, "layout.north"), `<!doctype html><html><head></head><body><slot /></body></html>`)
	writeRouteTestFile(t, filepath.Join(routes, "layout.north.go"), loaderSidecar("routes", "Layout"))
	writeRouteTestFile(t, filepath.Join(routes, "page.north"), `<p>Home</p>`)
	writeRouteTestFile(t, filepath.Join(routes, "page.north.go"), loaderSidecar("routes", "Page"))
	writeRouteTestFile(t, filepath.Join(api, "events", "route.go"), `package events
import "github.com/JohnKinyanjui/northframe/pkg/web"
func GET(ctx *web.Context) error { return nil }
func WEBSOCKET(ctx *web.Context, socket *web.Socket) error { return nil }
`)

	_, err := BuildProject(routes, api, "routesgen", "example.test/app/routes", "example.test/app/routes/api")
	if err == nil || !strings.Contains(err.Error(), "cannot declare both GET and WEBSOCKET") {
		t.Fatalf("BuildProject error = %v", err)
	}
}

func TestBuildProjectRejectsInvalidWebSocketSignature(t *testing.T) {
	appRoot := t.TempDir()
	routes := filepath.Join(appRoot, "routes")
	api := filepath.Join(routes, "api")
	writeRouteTestFile(t, filepath.Join(routes, "layout.north"), `<!doctype html><html><head></head><body><slot /></body></html>`)
	writeRouteTestFile(t, filepath.Join(routes, "layout.north.go"), loaderSidecar("routes", "Layout"))
	writeRouteTestFile(t, filepath.Join(routes, "page.north"), `<p>Home</p>`)
	writeRouteTestFile(t, filepath.Join(routes, "page.north.go"), loaderSidecar("routes", "Page"))
	writeRouteTestFile(t, filepath.Join(api, "events", "route.go"), `package events
func WEBSOCKET() error { return nil }
`)

	_, err := BuildProject(routes, api, "routesgen", "example.test/app/routes", "example.test/app/routes/api")
	if err == nil || !strings.Contains(err.Error(), "func WEBSOCKET(*web.Context, *web.Socket) error") {
		t.Fatalf("BuildProject error = %v", err)
	}
}

func TestBuildProjectRejectsInvalidAPIHandlerSignature(t *testing.T) {
	appRoot := t.TempDir()
	routes := filepath.Join(appRoot, "routes")
	api := filepath.Join(routes, "api")
	writeRouteTestFile(t, filepath.Join(routes, "layout.north"), `<!doctype html><html><body><slot /></body></html>`)
	writeRouteTestFile(t, filepath.Join(routes, "layout.north.go"), loaderSidecar("routes", "Layout"))
	writeRouteTestFile(t, filepath.Join(routes, "page.north"), `<p>Home</p>`)
	writeRouteTestFile(t, filepath.Join(routes, "page.north.go"), loaderSidecar("routes", "Page"))
	writeRouteTestFile(t, filepath.Join(api, "users", "get.go"), `package users
func GET() error { return nil }
`)
	_, err := BuildProject(routes, api, "routesgen", "example.test/app/routes", "example.test/app/routes/api")
	if err == nil || !strings.Contains(err.Error(), "func GET(*web.Context) error") {
		t.Fatalf("BuildProject error = %v", err)
	}
}

func TestBuildRoutesCreatesIndependentTypedComponents(t *testing.T) {
	appRoot := t.TempDir()
	root := filepath.Join(appRoot, "routes")
	writeRouteTestFile(t, filepath.Join(root, "layout.north"), `<!doctype html><html><head></head><body><slot /></body></html>`)
	writeRouteTestFile(t, filepath.Join(root, "layout.north.go"), loaderSidecar("routes", "Layout"))
	writeRouteTestFile(t, filepath.Join(root, "page.north"), `<Panel Title={Props.Title}><p>Slotted content</p></Panel>`)
	writeRouteTestFile(t, filepath.Join(root, "page.north.go"), `package routes
import "github.com/JohnKinyanjui/northframe/pkg/web"
type PageProps struct { Title string }
func Page(*web.Context) (PageProps, error) { return PageProps{Title: "Home"}, nil }
`)
	writeRouteTestFile(t, filepath.Join(appRoot, "components", "panel.north"), `<script context="props">
Title string
</script>
<script lang="ts">
let open: boolean = false;
let label: string = props.Title;
function toggle(): void { open = !open; }
</script>
<section><button on:click={toggle}>{Props.Title}</button><div show={#open}><slot /></div></section>`)

	build, err := BuildRoutes(root, "routesgen", "example.test/app/routes")
	if err != nil {
		t.Fatal(err)
	}
	component := string(build.Files["component_panel_generated.go"])
	page := string(build.Files["home_page_generated.go"])
	contracts := string(build.Files["northframe_contracts_generated.ts"])
	router := string(build.Files["router_generated.go"])
	for _, expected := range []string{
		`type PanelProps struct`, `Content web.Fragment`, `data-north-scope=\"panel\"`,
		`RenderPanel(w, PanelProps{`, `Title: props.Title`, `Slotted content`,
		`type PanelProps =`, `readonly Title: string`, `document.querySelectorAll('[data-north-scope=\"panel\"]')`,
		`const props = readProps(root, \"panel\")`, `mountComponent({`, `}, root)`,
	} {
		combined := component + page + contracts + router
		if !strings.Contains(combined, expected) {
			t.Errorf("generated component output does not contain %q\n%s", expected, combined)
		}
	}
	if len(router) == 0 {
		t.Fatal("router was not generated")
	}
}

func TestBuildRoutesGeneratesRoutePropsFromTemplates(t *testing.T) {
	appRoot := t.TempDir()
	root := filepath.Join(appRoot, "routes")
	writeRouteTestFile(t, filepath.Join(root, "layout.north"), `<!doctype html><html><head><title>{Props.Title}</title></head><body><h1>{Props.Title}</h1><slot /></body></html>`)
	writeRouteTestFile(t, filepath.Join(root, "layout.north.go"), `package routes
import "github.com/JohnKinyanjui/northframe/pkg/web"
func Layout(*web.Context) (LayoutProps, error) { return LayoutProps{Title: "Generated"}, nil }
`)
	writeRouteTestFile(t, filepath.Join(root, "page.north"), `<script context="props">
import models "example.test/app/viewmodels"
Dashboard models.Dashboard
</script><p>{Props.Dashboard.Name}</p>`)
	writeRouteTestFile(t, filepath.Join(root, "page.north.go"), `package routes
import (
  models "example.test/app/viewmodels"
  "github.com/JohnKinyanjui/northframe/pkg/web"
)
func Page(*web.Context) (PageProps, error) { return PageProps{Dashboard: models.Dashboard{}}, nil }
`)

	build, err := BuildRoutes(root, "routesgen", "example.test/app/routes")
	if err != nil {
		t.Fatal(err)
	}
	propsPath := filepath.Join("root", routePropsFilename)
	generated := string(build.Files[propsPath])
	for _, expected := range []string{
		`type LayoutProps struct`, `Title string`, `type PageProps struct`,
		`models "example.test/app/viewmodels"`, `Dashboard models.Dashboard`,
	} {
		if !strings.Contains(generated, expected) {
			t.Errorf("generated route props do not contain %q\n%s", expected, generated)
		}
	}
}

func TestBuildRoutesRequiresContractForNestedInferredProp(t *testing.T) {
	root := t.TempDir()
	writeRouteTestFile(t, filepath.Join(root, "layout.north"), `<html><body><slot /></body></html>`)
	writeRouteTestFile(t, filepath.Join(root, "layout.north.go"), `package routes
import "github.com/JohnKinyanjui/northframe/pkg/web"
func Layout(*web.Context) (LayoutProps, error) { return LayoutProps{}, nil }
`)
	writeRouteTestFile(t, filepath.Join(root, "page.north"), `<p>{Props.Dashboard.Name}</p>`)
	writeRouteTestFile(t, filepath.Join(root, "page.north.go"), `package routes
import "github.com/JohnKinyanjui/northframe/pkg/web"
func Page(*web.Context) (PageProps, error) { return PageProps{}, nil }
`)

	_, err := BuildRoutes(root, "routesgen", "example.test/app/routes")
	if err == nil || !strings.Contains(err.Error(), `interface Props`) {
		t.Fatalf("BuildRoutes error = %v", err)
	}
}

func TestBuildRoutesRejectsInvalidSidecarLoader(t *testing.T) {
	root := t.TempDir()
	writeRouteTestFile(t, filepath.Join(root, "layout.north"), `<!doctype html><html><head></head><body><slot /></body></html>`)
	writeRouteTestFile(t, filepath.Join(root, "layout.north.go"), `package routes
type LayoutProps struct{}
func Layout() LayoutProps { return LayoutProps{} }
`)
	writeRouteTestFile(t, filepath.Join(root, "page.north"), `<p>Home</p>`)
	writeRouteTestFile(t, filepath.Join(root, "page.north.go"), loaderSidecar("routes", "Page"))
	_, err := BuildRoutes(root, "routesgen", "example.test/app/routes")
	if err == nil || !strings.Contains(err.Error(), "func Layout(*web.Context) (LayoutProps, error)") {
		t.Fatalf("BuildRoutes() error = %v", err)
	}
}

func TestBuildRoutesRejectsRetiredNFExtension(t *testing.T) {
	root := t.TempDir()
	writeRouteTestFile(t, filepath.Join(root, "page.nf"), `<p>Legacy</p>`)
	_, err := BuildRoutes(root, "routesgen", "example.test/app/routes")
	if err == nil || !strings.Contains(err.Error(), "rename it to .north") {
		t.Fatalf("BuildRoutes() error = %v, want migration guidance", err)
	}
}

func loaderSidecar(packageName, kind string) string {
	return "package " + packageName + `
import "github.com/JohnKinyanjui/northframe/pkg/web"
type ` + kind + `Props struct{}
func ` + kind + `(*web.Context) (` + kind + `Props, error) { return ` + kind + `Props{}, nil }
`
}

func writeRouteTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
