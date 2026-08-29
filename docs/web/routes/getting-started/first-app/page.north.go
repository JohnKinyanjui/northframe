package first_app

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/getting-started/first-app"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document("Your first app", "Create, run, and understand a complete Northframe application from its first route.", "Start here", "getting-started/first-app.md")}, nil
}
