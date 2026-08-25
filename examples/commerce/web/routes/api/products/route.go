package products

import (
	"errors"
	"net/http"

	"northframe.dev/northframe/examples/commerce/services/commerce"
	"northframe.dev/northframe/pkg/web"
)

type createProductRequest struct {
	SKU          string `json:"sku"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	Stock        int32  `json:"stock"`
	ReorderPoint int32  `json:"reorder_point"`
	PriceCents   int64  `json:"price_cents"`
}

// GET serves GET /api/products.
func GET(ctx *web.Context) error {
	store := web.MustUse[commerce.Store](ctx)
	items, err := commerce.GetInventory(ctx.StdContext(), store, ctx.Query("q"))
	if err != nil {
		return web.Error(http.StatusInternalServerError, "unable to list products", err)
	}
	return ctx.JSON(http.StatusOK, map[string]any{"products": items})
}

// POST serves POST /api/products.
func POST(ctx *web.Context) error {
	var request createProductRequest
	if err := ctx.DecodeJSON(&request); err != nil {
		return web.BadRequest("invalid product JSON", err)
	}
	store := web.MustUse[commerce.Store](ctx)
	product, err := commerce.CreateProduct(ctx.StdContext(), store, commerce.ProductInput{
		SKU: request.SKU, Name: request.Name, Category: request.Category,
		Stock: request.Stock, ReorderPoint: request.ReorderPoint, PriceCents: request.PriceCents,
	})
	if errors.Is(err, commerce.ErrInvalidProduct) {
		return web.BadRequest("invalid product", err)
	}
	if err != nil {
		return web.Error(http.StatusInternalServerError, "unable to create product", err)
	}
	return ctx.JSON(http.StatusCreated, product)
}
