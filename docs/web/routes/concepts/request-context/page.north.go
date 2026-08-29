package request_context

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/concepts/request-context"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document("Request context", "Access parameters, forms, cancellation, databases, and application services through web.Context.", "Core concepts", "concepts/request-context.md")}, nil
}
