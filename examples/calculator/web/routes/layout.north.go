package routes

import "github.com/JohnKinyanjui/northframe/pkg/web"

type LayoutProps struct {
	Title string
}

func Layout(*web.Context) (LayoutProps, error) {
	return LayoutProps{Title: "Northframe Calculator"}, nil
}
