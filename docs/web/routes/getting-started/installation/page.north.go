package installation

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/getting-started/installation"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document("Installation", "Install the Northframe CLI and create a project without adding Node or Deno to your application.", "Start here", "getting-started/installation.md")}, nil
}
