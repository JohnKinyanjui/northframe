package database

import (
	generated "github.com/JohnKinyanjui/northframe/docs/.generated/routes/guides/database"
	"github.com/JohnKinyanjui/northframe/docs/internal/site"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(*web.Context) (generated.PageProps, error) {
	return generated.PageProps{Page: site.Document("Database and sqlc", "Connect PostgreSQL, MySQL, or SQLite and generate type-safe query code with sqlc.", "Guides", "guides/database.md")}, nil
}
