package commerce

import (
	"context"

	db "github.com/JohnKinyanjui/northframe/examples/commerce/internal/db/generated"
)

type Store interface {
	GetCommerceStats(context.Context) (db.GetCommerceStatsRow, error)
	ListOrders(context.Context) ([]db.ListOrdersRow, error)
	ListInventory(context.Context) ([]db.ListInventoryRow, error)
	CreateProduct(context.Context, db.CreateProductParams) (db.Product, error)
}

type ProductInput struct {
	SKU          string
	Name         string
	Category     string
	Stock        int32
	ReorderPoint int32
	PriceCents   int64
}
