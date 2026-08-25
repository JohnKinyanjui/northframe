package routes

import (
	generated "northframe.dev/northframe/examples/commerce/.generated/routes/root"
	"northframe.dev/northframe/examples/commerce/services/commerce"
	"northframe.dev/northframe/pkg/web"
)

func Page(ctx *web.Context) (generated.PageProps, error) {
	store := web.MustUse[commerce.Store](ctx)
	dashboard, err := commerce.GetDashboard(ctx.StdContext(), store)
	if err != nil {
		return generated.PageProps{}, err
	}
	return generated.PageProps{Dashboard: dashboard}, nil
}
