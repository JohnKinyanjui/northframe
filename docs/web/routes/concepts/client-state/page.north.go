package client_state

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/concepts/client-state"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document("Client state", "Use small TypeScript islands for interaction while Go continues to own server state and HTML.", "Core concepts", "concepts/client-state.md")}, nil
}
