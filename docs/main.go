package main

import (
	"log"
	"net/http"
	"os"

	frameworkassets "github.com/JohnKinyanjui/northframe/assets"
	"github.com/JohnKinyanjui/northframe/docs/.generated/routes"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	app := web.New()
	app.Handle("GET /logo.png", web.PublicHandler(map[string]web.Asset{
		"logo.png": web.NewAsset(frameworkassets.Logo, "image/png"),
	}))
	routes.Register(app)
	log.Printf("Northframe documentation running on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, app))
}
