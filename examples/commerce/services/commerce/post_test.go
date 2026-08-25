package commerce

import (
	"context"
	"errors"
	"testing"

	db "northframe.dev/northframe/examples/commerce/internal/db/generated"
)

type productStore struct {
	created db.CreateProductParams
	calls   int
}

func (*productStore) GetCommerceStats(context.Context) (db.GetCommerceStatsRow, error) {
	return db.GetCommerceStatsRow{}, nil
}

func (*productStore) ListOrders(context.Context) ([]db.ListOrdersRow, error) {
	return nil, nil
}

func (*productStore) ListInventory(context.Context) ([]db.ListInventoryRow, error) {
	return nil, nil
}

func (store *productStore) CreateProduct(_ context.Context, input db.CreateProductParams) (db.Product, error) {
	store.created = input
	store.calls++
	return db.Product{
		ID: 42, Sku: input.Sku, Name: input.Name, Category: input.Category,
		Stock: input.Stock, ReorderPoint: input.ReorderPoint, PriceCents: input.PriceCents,
	}, nil
}

func TestCreateProductNormalizesAndPersistsInput(t *testing.T) {
	store := &productStore{}
	product, err := CreateProduct(context.Background(), store, ProductInput{
		SKU: "  bowl-42 ", Name: "  Serving Bowl  ", Category: "  Tableware ",
		Stock: 12, ReorderPoint: 4, PriceCents: 14950,
	})
	if err != nil {
		t.Fatal(err)
	}
	if store.calls != 1 {
		t.Fatalf("CreateProduct calls = %d, want 1", store.calls)
	}
	if store.created.Sku != "BOWL-42" || store.created.Name != "Serving Bowl" || store.created.Category != "Tableware" {
		t.Fatalf("normalized input = %#v", store.created)
	}
	if product.ID != 42 || product.PriceCents != 14950 {
		t.Fatalf("created product = %#v", product)
	}
}

func TestCreateProductRejectsInvalidInputBeforeDatabase(t *testing.T) {
	tests := []struct {
		name  string
		input ProductInput
	}{
		{name: "missing sku", input: ProductInput{Name: "Bowl", Category: "Table"}},
		{name: "missing name", input: ProductInput{SKU: "B-1", Category: "Table"}},
		{name: "missing category", input: ProductInput{SKU: "B-1", Name: "Bowl"}},
		{name: "negative stock", input: ProductInput{SKU: "B-1", Name: "Bowl", Category: "Table", Stock: -1}},
		{name: "negative reorder point", input: ProductInput{SKU: "B-1", Name: "Bowl", Category: "Table", ReorderPoint: -1}},
		{name: "negative price", input: ProductInput{SKU: "B-1", Name: "Bowl", Category: "Table", PriceCents: -1}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &productStore{}
			_, err := CreateProduct(context.Background(), store, test.input)
			if !errors.Is(err, ErrInvalidProduct) {
				t.Fatalf("CreateProduct error = %v, want ErrInvalidProduct", err)
			}
			if store.calls != 0 {
				t.Fatalf("database called %d times for invalid input", store.calls)
			}
		})
	}
}
