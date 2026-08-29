package north_syntax

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/reference/north-syntax"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document(
		".north syntax",
		"The complete reference for Go props, server rendering, control flow, components, layouts, and TypeScript browser state.",
		"Reference",
		"reference/north-syntax.md",
	)}, nil
}
