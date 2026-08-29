package routes

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/root"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document("Northframe", "Build server-rendered applications with Go at the center and TypeScript only where the browser needs it.", "Introduction", "index.md")}, nil
}
