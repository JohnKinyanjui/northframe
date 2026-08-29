package commerce

import (
	"context"
	"fmt"
	"strings"

	"github.com/JohnKinyanjui/northframe/examples/commerce/internal/viewmodels"
)

func GetDashboard(ctx context.Context, store Store) (viewmodels.Dashboard, error) {
	stats, err := store.GetCommerceStats(ctx)
	if err != nil {
		return viewmodels.Dashboard{}, fmt.Errorf("get commerce stats: %w", err)
	}
	rows, err := store.ListOrders(ctx)
	if err != nil {
		return viewmodels.Dashboard{}, fmt.Errorf("list orders: %w", err)
	}
	result := viewmodels.Dashboard{
		ProcessingOrders: stats.ProcessingOrders,
		LowStock:         stats.LowStock,
		Gross:            formatMoney(stats.GrossCents),
		TotalOrders:      stats.TotalOrders,
		Orders:           make([]viewmodels.Order, 0, len(rows)),
	}
	for _, row := range rows {
		result.Orders = append(result.Orders, viewmodels.Order{
			Reference: row.Reference, Customer: row.CustomerName, Total: formatMoney(row.TotalCents),
			Status: row.Status, Destination: row.Destination, PlacedAt: row.PlacedAt,
		})
	}
	return result, nil
}

func GetInventory(ctx context.Context, store Store, search string) ([]viewmodels.InventoryItem, error) {
	rows, err := store.ListInventory(ctx)
	if err != nil {
		return nil, fmt.Errorf("list inventory: %w", err)
	}
	search = strings.ToLower(strings.TrimSpace(search))
	result := make([]viewmodels.InventoryItem, 0, len(rows))
	for _, row := range rows {
		if search != "" && !strings.Contains(strings.ToLower(row.Name+" "+row.Sku+" "+row.Category), search) {
			continue
		}
		result = append(result, viewmodels.InventoryItem{
			SKU: row.Sku, Name: row.Name, Category: row.Category, Stock: row.Stock,
			ReorderPoint: row.ReorderPoint, Price: formatMoney(row.PriceCents), Health: row.Health,
		})
	}
	return result, nil
}
