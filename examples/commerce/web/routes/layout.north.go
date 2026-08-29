package routes

import (
	generated "github.com/JohnKinyanjui/northframe/examples/commerce/.generated/routes/root"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Layout(*web.Context) (generated.LayoutProps, error) {
	return generated.LayoutProps{Title: "Relay Commerce", Operator: "M. Okafor"}, nil
}

func LayoutMiddleware() []web.Middleware {
	return []web.Middleware{web.CSRF()}
}
