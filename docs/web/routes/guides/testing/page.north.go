package testing

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/guides/testing"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document("Testing", "Test services, generated routes, forms, APIs, sessions, and browser behavior with normal Go tooling.", "Guides", "guides/testing.md")}, nil
}
