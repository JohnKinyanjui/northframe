package background_services

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/guides/background-services"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document("Jobs, mail, and cache", "Move background work, email, and cached data behind small typed services and production-ready boundaries.", "Guides", "guides/background-services.md")}, nil
}
