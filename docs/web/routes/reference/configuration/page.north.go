package configuration

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/reference/configuration"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document("northframe.toml", "Understand Northframe's focused project manifest and reproducible browser dependency lockfile.", "Reference", "reference/configuration.md")}, nil
}
