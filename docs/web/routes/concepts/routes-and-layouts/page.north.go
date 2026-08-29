package routes_and_layouts

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/concepts/routes-and-layouts"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document("Routes and layouts", "Turn the filesystem into typed pages, nested layouts, dynamic parameters, and API endpoints.", "Core concepts", "concepts/routes-and-layouts.md")}, nil
}
