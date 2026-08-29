package editor

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/reference/editor"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document("Editor and LSP", "Configure VS Code formatting, diagnostics, completion, navigation, and embedded Go and TypeScript intelligence.", "Reference", "reference/editor.md")}, nil
}
