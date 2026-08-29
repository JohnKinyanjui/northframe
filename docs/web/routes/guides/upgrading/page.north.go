package upgrading

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/guides/upgrading"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document("Upgrading", "Update Northframe-managed output safely while preserving application-owned code and configuration.", "Guides", "guides/upgrading.md")}, nil
}
