package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

var invalidModuleCharacter = regexp.MustCompile(`[^A-Za-z0-9._~/-]+`)

type scaffoldFile struct {
	path     string
	contents string
}

func createProject(arguments []string) error {
	flags := flag.NewFlagSet("create", flag.ContinueOnError)
	module := flags.String("module", "", "Go module path (defaults to the directory name)")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return errors.New("usage: north create [--module MODULE] DIRECTORY")
	}

	target, err := filepath.Abs(flags.Arg(0))
	if err != nil {
		return fmt.Errorf("resolve project directory: %w", err)
	}
	if info, statErr := os.Stat(target); statErr == nil && !info.IsDir() {
		return fmt.Errorf("create target %s is not a directory", target)
	} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("inspect project directory: %w", statErr)
	}

	modulePath := strings.TrimSpace(*module)
	if modulePath == "" {
		modulePath = defaultModulePath(filepath.Base(target))
	}
	if modulePath == "" {
		return errors.New("cannot derive a Go module name; pass --module")
	}
	projectName := displayProjectName(filepath.Base(target))
	files := projectScaffold(modulePath, projectName)
	if err := preflightScaffold(target, files); err != nil {
		return err
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return fmt.Errorf("create project directory: %w", err)
	}
	for _, file := range files {
		path := filepath.Join(target, filepath.FromSlash(file.path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create %s: %w", file.path, err)
		}
		if err := os.WriteFile(path, []byte(file.contents), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", file.path, err)
		}
	}

	options := projectOptions{
		routes:      filepath.Join(target, "web", "routes"),
		generated:   filepath.Join(target, ".generated", "routes"),
		packageName: "routes",
	}
	count, err := generateProject(options)
	if err != nil {
		return fmt.Errorf("project files were created, but initial generation failed: %w", err)
	}
	fmt.Printf("created %s with %d compiled route(s)\n", target, count)
	fmt.Printf("next: cd %s && north run\n", target)
	return nil
}

func preflightScaffold(root string, files []scaffoldFile) error {
	var conflicts []string
	for _, file := range files {
		path := filepath.Join(root, filepath.FromSlash(file.path))
		existing, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			conflicts = append(conflicts, file.path)
			continue
		}
		if string(existing) != file.contents {
			conflicts = append(conflicts, file.path)
		}
	}
	if len(conflicts) == 0 {
		return nil
	}
	sort.Strings(conflicts)
	return fmt.Errorf("refusing to overwrite existing project file(s): %s", strings.Join(conflicts, ", "))
}

func defaultModulePath(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, " ", "-")
	return strings.Trim(invalidModuleCharacter.ReplaceAllString(name, "-"), "-./")
}

func displayProjectName(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool { return r == '-' || r == '_' || unicode.IsSpace(r) })
	for index, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(strings.ToLower(part))
		runes[0] = unicode.ToUpper(runes[0])
		parts[index] = string(runes)
	}
	if len(parts) == 0 {
		return "Northframe"
	}
	return strings.Join(parts, " ")
}

func projectScaffold(modulePath, projectName string) []scaffoldFile {
	return []scaffoldFile{
		{path: "go.mod", contents: fmt.Sprintf("module %s\n\ngo 1.27\n\nrequire northframe.dev/northframe v0.0.0\n", modulePath)},
		{path: "northframe.toml", contents: "[client]\nsource = 'web/client'\n\n[client.dependencies]\n"},
		{path: ".gitignore", contents: ".env\n.generated/\n.northframe/\napp\n"},
		{path: "main.go", contents: fmt.Sprintf(`package main

import (
	"log"
	"net/http"
	"os"

	"%s/.generated/routes"
	"northframe.dev/northframe/pkg/web"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	app := web.New()
	routes.Register(app)
	log.Printf("%s running on http://localhost:%%s", port)
	log.Fatal(http.ListenAndServe(":"+port, app))
}
`, modulePath, projectName)},
		{path: "web/routes/layout.north", contents: `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>{Props.Title}</title>
    <link rel="stylesheet" href="/_northframe/app.css" />
  </head>
  <body class="min-h-screen bg-stone-950 text-stone-100 antialiased">
    <slot />
  </body>
</html>
`},
		{path: "web/routes/layout.north.go", contents: fmt.Sprintf(`package routes

import (
	generated "%s/.generated/routes/root"
	"northframe.dev/northframe/pkg/web"
)

func Layout(*web.Context) (generated.LayoutProps, error) {
	return generated.LayoutProps{Title: %q}, nil
}
`, modulePath, projectName)},
		{path: "web/routes/page.north", contents: `<main class="mx-auto flex min-h-screen max-w-5xl items-center px-6 py-20">
  <section class="max-w-3xl">
    <p class="mb-5 text-sm font-semibold uppercase tracking-[0.24em] text-orange-400">Northframe application</p>
    <h1 class="text-5xl font-black tracking-tight sm:text-7xl">{Props.ProjectName} is ready.</h1>
    <p class="mt-7 max-w-2xl text-lg leading-8 text-stone-400">Native Go SSR, typed browser state, structured routes, and one deployment binary.</p>
    <div class="mt-10 flex flex-wrap gap-3">
      <a class="rounded-full bg-orange-500 px-6 py-3 font-bold text-stone-950 hover:bg-orange-400" href="/api/health">Check API</a>
      <span class="rounded-full border border-stone-700 px-6 py-3 text-stone-300">Edit web/routes/page.north</span>
    </div>
  </section>
</main>
`},
		{path: "web/routes/page.north.go", contents: fmt.Sprintf(`package routes

import (
	generated "%s/.generated/routes/root"
	"northframe.dev/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{ProjectName: %q}, nil
}
`, modulePath, projectName)},
		{path: "web/routes/api/health/route.go", contents: `package health

import "northframe.dev/northframe/pkg/web"

func GET(ctx *web.Context) error {
	return ctx.JSON(200, map[string]string{"status": "ok"})
}
`},
	}
}
