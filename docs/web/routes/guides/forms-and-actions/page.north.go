package forms_and_actions

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/guides/forms-and-actions"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document("Forms and actions", "Handle standard browser submissions with typed Go actions, validation, redirects, and loading states.", "Guides", "guides/forms-and-actions.md")}, nil
}
