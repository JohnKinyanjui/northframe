package errors

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/reference/errors"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document("Error handling", "Understand safe HTTP errors today and the direction for filesystem-defined Northframe error pages.", "Reference", "reference/errors.md")}, nil
}
