package api_routes

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/guides/api-routes"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document("API routes", "Build typed JSON endpoints and WebSockets inside the same filesystem route tree and Go server.", "Guides", "guides/api-routes.md")}, nil
}
