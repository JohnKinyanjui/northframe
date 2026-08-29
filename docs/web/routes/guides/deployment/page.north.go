package deployment

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/guides/deployment"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document("Deployment", "Validate and ship a production Northframe application as one executable or container image.", "Guides", "guides/deployment.md")}, nil
}
