package authentication

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/guides/authentication"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document("Authentication", "Create opaque server sessions, protect routes, and enforce permissions without replacing your user model.", "Guides", "guides/authentication.md")}, nil
}
