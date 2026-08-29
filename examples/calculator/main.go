package main

import (
	"log"
	"net/http"
	"os"

	"github.com/JohnKinyanjui/northframe/examples/calculator/.generated/routes"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	app := web.New()
	routes.Register(app)

	log.Printf("Northframe calculator running on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, app))
}
