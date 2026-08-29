package content

import (
	"bytes"
	"strings"
	"testing"

	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func TestRenderLoadsEmbeddedDocumentation(t *testing.T) {
	value, err := Render("index.md")
	if err != nil {
		t.Fatal(err)
	}
	var rendered bytes.Buffer
	if err := web.WriteHTML(&rendered, value); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered.String(), "Northframe is a Go-first") {
		t.Fatalf("unexpected documentation output: %s", rendered.String())
	}
}
