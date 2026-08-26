package live

import (
	"time"

	"northframe.dev/northframe/examples/commerce/internal/viewmodels"
	"northframe.dev/northframe/examples/commerce/services/commerce"
	"northframe.dev/northframe/pkg/web"
)

type inventoryRequest struct {
	Search string `json:"search"`
}

type inventoryResponse struct {
	Products []viewmodels.InventoryItem `json:"products"`
}

// WebSocketOptions keeps the example safe under slow or abusive clients while
// retaining Northframe's default same-origin verification.
func WebSocketOptions() web.SocketOptions {
	return web.SocketOptions{
		ReadLimit:      8 << 10,
		ReadTimeout:    75 * time.Second,
		WriteTimeout:   5 * time.Second,
		MaxConnections: 250,
	}
}

// WEBSOCKET serves /api/products/live. Send {"search":"..."} to receive the
// current filtered PostgreSQL inventory without introducing a second server.
func WEBSOCKET(ctx *web.Context, socket *web.Socket) error {
	store := web.MustUse[commerce.Store](ctx)
	for {
		var request inventoryRequest
		if err := socket.ReadJSON(ctx.StdContext(), &request); err != nil {
			return err
		}
		products, err := commerce.GetInventory(ctx.StdContext(), store, request.Search)
		if err != nil {
			return err
		}
		if err := socket.WriteJSON(ctx.StdContext(), inventoryResponse{Products: products}); err != nil {
			return err
		}
	}
}
