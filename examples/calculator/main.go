package main

import (
	"log"
	"net/http"
	"os"

	"northframe.dev/northframe/examples/calculator/.generated/routes"
	"northframe.dev/northframe/pkg/web"
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
