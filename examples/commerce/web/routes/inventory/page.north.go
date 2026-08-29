package inventory

import (
	"errors"
	"net/http"

	generated "github.com/JohnKinyanjui/northframe/examples/commerce/.generated/routes/inventory"
	"github.com/JohnKinyanjui/northframe/examples/commerce/services/commerce"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

type AddProductInput struct {
	Name         string  `form:"name" label:"Product name" validate:"required,min=2,max=120"`
	SKU          string  `form:"sku" label:"SKU" validate:"required,min=2,max=40"`
	Category     string  `form:"category" validate:"required,min=2,max=80"`
	Stock        int32   `form:"stock" label:"Opening stock" validate:"required,min=0"`
	ReorderPoint int32   `form:"reorder_point" label:"Reorder point" validate:"required,min=0"`
	Price        float64 `form:"price" label:"Retail price" validate:"required,min=0"`
}

func PageActions() []web.Action {
	return []web.Action{web.PostForm("/inventory/products", addProduct)}
}

func addProduct(ctx *web.Context, input AddProductInput) (web.ActionResult, error) {
	store := web.MustUse[commerce.Store](ctx)
	_, err := commerce.CreateProduct(ctx.StdContext(), store, commerce.ProductInput{
		SKU: input.SKU, Name: input.Name, Category: input.Category,
		Stock: input.Stock, ReorderPoint: input.ReorderPoint, PriceCents: int64(input.Price*100 + 0.5),
	})
	if errors.Is(err, commerce.ErrInvalidProduct) {
		return web.ActionInvalid("Please correct the product details.", web.FieldErrors{
			"form": "The product could not be validated.",
		}), nil
	}
	if err != nil {
		return web.ActionResult{}, web.Error(http.StatusInternalServerError, "unable to add product", err)
	}
	return web.ActionRedirect("/inventory", http.StatusSeeOther), nil
}

func Page(ctx *web.Context) (generated.PageProps, error) {
	search := ctx.Query("q")
	store := web.MustUse[commerce.Store](ctx)
	items, err := commerce.GetInventory(ctx.StdContext(), store, search)
	if err != nil {
		return generated.PageProps{}, err
	}
	return generated.PageProps{Items: items, Query: search}, nil
}
