package admin

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/guides/admin"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document("Internal admin", "Mount a permission-aware internal administration UI over application services and typed resource definitions.", "Guides", "guides/admin.md")}, nil
}
