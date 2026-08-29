package client_dependencies

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/guides/client-dependencies"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document("Browser packages", "Add TypeScript libraries through Northframe without adopting package.json or a Node runtime.", "Guides", "guides/client-dependencies.md")}, nil
}
