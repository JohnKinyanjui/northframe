package site

import (
	"github.com/JohnKinyanjui/northframe/docs/content"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

type NavItem struct {
	Title string
	Href  string
}

type NavGroup struct {
	Title string
	Items []NavItem
}

var Navigation = []NavGroup{
	{Title: "Start here", Items: []NavItem{
		{Title: "Introduction", Href: "/"},
		{Title: "Installation", Href: "/getting-started/installation"},
		{Title: "Your first app", Href: "/getting-started/first-app"},
		{Title: "Project structure", Href: "/getting-started/project-structure"},
	}},
	{Title: "Core concepts", Items: []NavItem{
		{Title: "Routes and layouts", Href: "/concepts/routes-and-layouts"},
		{Title: "Props and rendering", Href: "/concepts/props"},
		{Title: "Components", Href: "/concepts/components"},
		{Title: "Client state", Href: "/concepts/client-state"},
		{Title: "Forms and actions", Href: "/guides/forms-and-actions"},
		{Title: "Request context", Href: "/concepts/request-context"},
	}},
	{Title: "Guides", Items: []NavItem{
		{Title: "Database and sqlc", Href: "/guides/database"},
		{Title: "API routes", Href: "/guides/api-routes"},
		{Title: "Authentication", Href: "/guides/authentication"},
		{Title: "Middleware and security", Href: "/guides/middleware-and-security"},
		{Title: "Testing", Href: "/guides/testing"},
		{Title: "Jobs, mail, and cache", Href: "/guides/background-services"},
		{Title: "Internal admin", Href: "/guides/admin"},
		{Title: "HTMX enhancement", Href: "/guides/htmx"},
		{Title: "Browser packages", Href: "/guides/client-dependencies"},
		{Title: "Deployment", Href: "/guides/deployment"},
		{Title: "Upgrading", Href: "/guides/upgrading"},
	}},
	{Title: "Reference", Items: []NavItem{
		{Title: ".north syntax", Href: "/reference/north-syntax"},
		{Title: "CLI", Href: "/reference/cli"},
		{Title: "northframe.toml", Href: "/reference/configuration"},
		{Title: "Editor and LSP", Href: "/reference/editor"},
		{Title: "Error handling", Href: "/reference/errors"},
	}},
}

type Page struct {
	Title       string
	Description string
	Section     string
	Content     web.SafeHTML
	EditHref    string
}

func Document(title, description, section, source string) Page {
	return Page{
		Title: title, Description: description, Section: section, Content: content.MustRender(source),
		EditHref: "https://github.com/JohnKinyanjui/northframe/edit/main/docs/content/" + source,
	}
}
