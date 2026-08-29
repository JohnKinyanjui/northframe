package cli

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/reference/cli"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document("CLI reference", "A practical reference for every current north command and its role in development or production.", "Reference", "reference/cli.md")}, nil
}
