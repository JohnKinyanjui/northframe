package middleware_and_security

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/guides/middleware-and-security"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document("Middleware and security", "Apply authentication, CSRF, headers, request locals, and structured request correlation safely.", "Guides", "guides/middleware-and-security.md")}, nil
}
