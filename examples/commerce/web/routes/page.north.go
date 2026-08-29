package routes

import (
	generated "github.com/JohnKinyanjui/northframe/examples/commerce/.generated/routes/root"
	"github.com/JohnKinyanjui/northframe/examples/commerce/services/commerce"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(ctx *web.Context) (generated.PageProps, error) {
	store := web.MustUse[commerce.Store](ctx)
	dashboard, err := commerce.GetDashboard(ctx.StdContext(), store)
	if err != nil {
		return generated.PageProps{}, err
	}
	return generated.PageProps{Dashboard: dashboard}, nil
}
