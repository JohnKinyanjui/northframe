package htmx

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/guides/htmx"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document("HTMX enhancement", "Use Northframe's embedded HTMX runtime for HTML-over-the-wire forms, polling, partial swaps, and navigation.", "Guides", "guides/htmx.md")}, nil
}
