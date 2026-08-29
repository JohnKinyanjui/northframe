package routes

import "github.com/JohnKinyanjui/northframe/pkg/web"

type PageProps struct {
	Title        string
	InitialValue string
}

func Page(*web.Context) (PageProps, error) {
	return PageProps{
		Title:        "Arithmetic, framed properly.",
		InitialValue: "0",
	}, nil
}
