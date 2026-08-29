package main

import (
	"errors"
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
	module, template, directory, err := parseCreateArguments(arguments)
	if err != nil {
		return err
	}

	target, err := filepath.Abs(directory)
	if err != nil {
		return fmt.Errorf("resolve project directory: %w", err)
	}
	if info, statErr := os.Stat(target); statErr == nil && !info.IsDir() {
		return fmt.Errorf("create target %s is not a directory", target)
	} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("inspect project directory: %w", statErr)
	}

	modulePath := strings.TrimSpace(module)
	if modulePath == "" {
		modulePath = defaultModulePath(filepath.Base(target))
	}
	if modulePath == "" {
		return errors.New("cannot derive a Go module name; pass --module")
	}
	projectName := displayProjectName(filepath.Base(target))
	files := projectScaffoldForTemplate(modulePath, projectName, template)
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

func parseCreateArguments(arguments []string) (module, template, directory string, err error) {
	template = "app"
	for index := 0; index < len(arguments); index++ {
		argument := arguments[index]
		switch {
		case strings.HasPrefix(argument, "--module="):
			module = strings.TrimSpace(strings.TrimPrefix(argument, "--module="))
		case argument == "--module":
			index++
			if index >= len(arguments) {
				return "", "", "", errors.New("--module requires a value")
			}
			module = arguments[index]
		case strings.HasPrefix(argument, "--template="):
			template = strings.TrimSpace(strings.TrimPrefix(argument, "--template="))
		case argument == "--template":
			index++
			if index >= len(arguments) {
				return "", "", "", errors.New("--template requires a value")
			}
			template = strings.TrimSpace(arguments[index])
		case strings.HasPrefix(argument, "-"):
			return "", "", "", fmt.Errorf("unknown create option %q", argument)
		case directory == "":
			directory = argument
		default:
			return "", "", "", errors.New("usage: north create DIRECTORY [--module MODULE] [--template app|docs]")
		}
	}
	if directory == "" {
		return "", "", "", errors.New("usage: north create DIRECTORY [--module MODULE] [--template app|docs]")
	}
	if template != "app" && template != "docs" {
		return "", "", "", fmt.Errorf("unknown project template %q; choose app or docs", template)
	}
	return module, template, directory, nil
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
	return projectScaffoldForTemplate(modulePath, projectName, "app")
}

func projectScaffoldForTemplate(modulePath, projectName, template string) []scaffoldFile {
	if template == "docs" {
		return docsProjectScaffold(modulePath, projectName)
	}
	return []scaffoldFile{
		{path: "go.mod", contents: fmt.Sprintf("module %s\n\ngo 1.27\n\nrequire github.com/JohnKinyanjui/northframe v0.0.0\n", modulePath)},
		{path: "northframe.toml", contents: "[client]\nsource = 'web/client'\n\n[client.dependencies]\n"},
		{path: ".gitignore", contents: ".env\n.generated/\n.northframe/\napp\n"},
		{path: "web/app.css", contents: "/* Project-wide styles. Northframe bundles this file after generated utilities. */\n"},
		{path: "main.go", contents: fmt.Sprintf(`package main

import (
	"log"
	"net/http"
	"os"

	"%s/.generated/routes"
	"github.com/JohnKinyanjui/northframe/pkg/web"
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
	"github.com/JohnKinyanjui/northframe/pkg/web"
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
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{ProjectName: %q}, nil
}
`, modulePath, projectName)},
		{path: "web/routes/error.north", contents: `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>${Props.Status} · Something went wrong</title>
  </head>
  <body class="min-h-screen bg-stone-950 text-stone-100 antialiased">
    <main class="mx-auto flex min-h-screen max-w-3xl items-center px-6 py-20">
      <section>
        <p class="text-sm font-bold uppercase tracking-[0.2em] text-orange-400">Error ${Props.Status}</p>
        <h1 class="mt-4 text-4xl font-black tracking-tight sm:text-6xl">${Props.Message}</h1>
        <p class="mt-5 text-stone-400">The request to <code>${Props.Path}</code> could not be completed.</p>
        <a class="mt-8 inline-flex rounded-full bg-orange-500 px-5 py-3 font-bold text-stone-950" href="/">Return home</a>
      </section>
    </main>
  </body>
</html>
`},
		{path: "web/routes/api/health/route.go", contents: `package health

import "github.com/JohnKinyanjui/northframe/pkg/web"

func GET(ctx *web.Context) error {
	return ctx.JSON(200, map[string]string{"status": "ok"})
}
`},
	}
}

func docsProjectScaffold(modulePath, projectName string) []scaffoldFile {
	return []scaffoldFile{
		{path: "go.mod", contents: fmt.Sprintf("module %s\n\ngo 1.27\n\nrequire github.com/JohnKinyanjui/northframe v0.0.0\n", modulePath)},
		{path: "northframe.toml", contents: "[client]\nsource = 'web/client'\n\n[client.dependencies]\n"},
		{path: ".gitignore", contents: ".env\n.generated/\n.northframe/\napp\n"},
		{path: "web/app.css", contents: "/* Project-wide styles. Northframe bundles this file after generated utilities. */\n"},
		{path: "main.go", contents: fmt.Sprintf(`package main

import (
	"log"
	"net/http"
	"os"

	"%s/.generated/routes"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" { port = "8000" }
	app := web.New()
	routes.Register(app)
	log.Printf(%q+" running on http://localhost:%%s", port)
	log.Fatal(http.ListenAndServe(":"+port, app))
}
`, modulePath, projectName+" documentation")},
		{path: "content/index.md", contents: fmt.Sprintf(`# %s

Welcome to your Northframe documentation site. Markdown stays pleasant to write, while Northframe owns routing, layouts, styling, and deployment.

## Start writing

- Edit `+"`content/index.md`"+` to change this article.
- Edit `+"`web/routes/layout.north`"+` to change the documentation shell.
- Add another route folder and Markdown file when you need a new page.

## One deployment binary

Run `+"`north build`"+` to embed the content and produce the same single Go executable as any other Northframe application.
`, projectName)},
		{path: "content/content.go", contents: `package content

import (
	"embed"

	northdocs "github.com/JohnKinyanjui/northframe/pkg/docs"
)

//go:embed *.md
var files embed.FS

func Index() northdocs.Document {
	document, err := northdocs.Load(files, "index.md")
	if err != nil { panic(err) }
	return document
}
`},
		{path: "web/routes/layout.north", contents: fmt.Sprintf(`<!doctype html>
<html lang="en">
  <head><meta charset="utf-8" /><meta name="viewport" content="width=device-width, initial-scale=1" /><title>${Props.Title}</title></head>
  <body class="min-h-screen bg-white text-slate-900 antialiased">
    <header class="sticky top-0 border-b border-slate-200 bg-white"><div class="mx-auto flex h-16 max-w-7xl items-center px-6"><a href="/" class="font-black">%s</a><span class="ml-2 text-sm text-slate-400">Docs</span></div></header>
    <div class="mx-auto flex max-w-7xl"><aside class="hidden min-h-[calc(100vh-4rem)] w-64 border-r border-slate-200 p-6 lg:block"><p class="text-xs font-bold uppercase tracking-widest text-slate-400">Start here</p><a href="/" class="mt-3 block border-l-2 border-orange-500 bg-orange-50 px-3 py-2 text-sm font-semibold text-orange-800">Introduction</a></aside><main class="min-w-0 flex-1"><slot /></main></div>
  </body>
</html>
`, projectName)},
		{path: "web/routes/layout.north.go", contents: fmt.Sprintf(`package routes

import (
	generated "%s/.generated/routes/root"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Layout(*web.Context) (generated.LayoutProps, error) { return generated.LayoutProps{Title: %q}, nil }
`, modulePath, projectName+" documentation")},
		{path: "web/routes/page.north", contents: `---
import northweb "github.com/JohnKinyanjui/northframe/pkg/web"

interface Props {
    Content northweb.SafeHTML
}
---

<article class="mx-auto max-w-3xl px-6 py-12">
  <template class="relative mt-12 mt-9 mt-6 mt-5 mt-4 mt-3 mt-0.5 flex hidden h-1.5 w-1.5 shrink-0 scroll-mt-24 space-y-2 gap-3 overflow-x-auto rounded-full rounded-r-xl rounded-xl rounded-md border border-t border-l-4 border-slate-200 border-sky-500 bg-slate-950 bg-slate-100 bg-sky-50 p-5 px-5 px-1.5 py-0.5 pl-1 pt-10 pb-5 text-[1.02rem] text-[0.9em] text-2xl text-xl text-sm font-extrabold font-bold font-semibold leading-8 leading-7 tracking-tight text-inherit text-slate-950 text-slate-700 text-slate-100 text-rose-700 text-sky-900 text-sky-700 underline no-underline decoration-sky-200 underline-offset-4 shadow-sm hover:text-sky-900"></template>
  <div>{html Props.Content}</div>
</article>
`},
		{path: "web/routes/page.north.go", contents: fmt.Sprintf(`package routes

import (
	generated "%s/.generated/routes/root"
	"%s/content"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) { return generated.PageProps{Content: content.Index().HTML}, nil }
`, modulePath, modulePath)},
		{path: "web/routes/error.north", contents: `<!doctype html><html lang="en"><head><meta charset="utf-8" /><meta name="viewport" content="width=device-width, initial-scale=1" /><title>${Props.Status} · Error</title></head><body class="min-h-screen bg-slate-950 text-white"><main class="mx-auto flex min-h-screen max-w-3xl items-center px-6"><section><p class="text-orange-400">Error ${Props.Status}</p><h1 class="mt-3 text-4xl font-black">${Props.Message}</h1><p class="mt-4 text-slate-400">${Props.Path}</p><a href="/" class="mt-7 inline-flex rounded-lg bg-orange-500 px-4 py-2 font-bold text-slate-950">Return home</a></section></main></body></html>
`},
	}
}
