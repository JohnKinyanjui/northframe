package commerce

import (
	"context"
	"errors"
	"fmt"
	"strings"

	db "github.com/JohnKinyanjui/northframe/examples/commerce/internal/db/generated"
)

var ErrInvalidProduct = errors.New("invalid product")

func CreateProduct(ctx context.Context, store Store, input ProductInput) (db.Product, error) {
	input.SKU = strings.ToUpper(strings.TrimSpace(input.SKU))
	input.Name = strings.TrimSpace(input.Name)
	input.Category = strings.TrimSpace(input.Category)
	if input.SKU == "" || input.Name == "" || input.Category == "" {
		return db.Product{}, fmt.Errorf("%w: sku, name, and category are required", ErrInvalidProduct)
	}
	if input.Stock < 0 || input.ReorderPoint < 0 || input.PriceCents < 0 {
		return db.Product{}, fmt.Errorf("%w: stock, reorder point, and price cannot be negative", ErrInvalidProduct)
	}
	product, err := store.CreateProduct(ctx, db.CreateProductParams{
		Sku:          input.SKU,
		Name:         input.Name,
		Category:     input.Category,
		Stock:        input.Stock,
		ReorderPoint: input.ReorderPoint,
		PriceCents:   input.PriceCents,
	})
	if err != nil {
		return db.Product{}, fmt.Errorf("create product: %w", err)
	}
	return product, nil
}
