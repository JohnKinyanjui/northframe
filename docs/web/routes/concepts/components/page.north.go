package components

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/concepts/components"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document("Components", "Extract reusable typed views, pass children through slots, and keep rendering native to Go.", "Core concepts", "concepts/components.md")}, nil
}
