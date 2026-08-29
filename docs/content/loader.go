// Package content embeds Northframe's documentation source.
package content

import (
	"embed"

	northdocs "github.com/JohnKinyanjui/northframe/pkg/docs"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

//go:embed *.md getting-started/*.md concepts/*.md guides/*.md reference/*.md
var files embed.FS

func Render(name string) (web.SafeHTML, error) {
	document, err := northdocs.Load(files, name)
	return document.HTML, err
}

func MustRender(name string) web.SafeHTML {
	result, err := Render(name)
	if err != nil {
		panic(err)
	}
	return result
}
