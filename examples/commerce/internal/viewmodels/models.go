package viewmodels

type Dashboard struct {
	ProcessingOrders int64
	LowStock         int64
	Gross            string
	TotalOrders      int64
	Orders           []Order
}

type Order struct {
	Reference   string
	Customer    string
	Total       string
	Status      string
	Destination string
	PlacedAt    string
}

type InventoryItem struct {
	SKU          string
	Name         string
	Category     string
	Stock        int32
	ReorderPoint int32
	Price        string
	Health       string
}

type Question struct {
	Category string
	Question string
	Answer   string
	Owner    string
}
