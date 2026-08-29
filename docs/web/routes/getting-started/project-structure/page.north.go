package project_structure

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/getting-started/project-structure"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document("Project structure", "Learn where routes, components, browser code, public files, and generated output belong.", "Start here", "getting-started/project-structure.md")}, nil
}
