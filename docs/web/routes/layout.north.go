package routes

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/root"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Layout(ctx *web.Context) (generated.LayoutProps, error) {
	path := "/"
	if ctx != nil && ctx.Request != nil && ctx.Request.URL != nil {
		path = ctx.Request.URL.Path
	}
	return generated.LayoutProps{
		Title: "Documentation", Path: path, Navigation: site.Navigation,
	}, nil
}
