package routes

import (
	generated "northframe.dev/northframe/examples/commerce/.generated/routes/root"
	"northframe.dev/northframe/pkg/web"
)

func Layout(*web.Context) (generated.LayoutProps, error) {
	return generated.LayoutProps{Title: "Relay Commerce", Operator: "M. Okafor"}, nil
}

func LayoutMiddleware() []web.Middleware {
	return []web.Middleware{web.CSRF()}
}
