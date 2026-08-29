package props

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/concepts/props"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document("Props and rendering", "Move typed Go data into escaped server-rendered HTML with an explicit, readable contract.", "Core concepts", "concepts/props.md")}, nil
}
