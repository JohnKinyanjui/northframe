package lsp

import (
	"strings"
	"testing"
)

func TestRefactorTargetRenamesInlinePropsEverywhere(t *testing.T) {
	text := "---\ninterface Props {\nTitle string\n}\n---\n<h1>{Props.Title}</h1><p>{Props.Title}</p>"
	offset := strings.Index(text, "Props.Title") + len("Props.") + 2
	target, ok := refactorTarget("file:///tmp/page.north", text, offset)
	if !ok || target.Name != "Title" || len(target.Occurrences) != 3 {
		t.Fatalf("target = %#v, %v", target, ok)
	}
}

func TestRefactorTargetScopesGoRangeVariable(t *testing.T) {
	text := "---\ninterface Props {\nItems []string\n}\n---\n{for item := range Props.Items}<p>{item}</p>{/for}<p>{item}</p>"
	offset := strings.Index(text, "<p>{item}") + len("<p>{") + 2
	target, ok := refactorTarget("file:///tmp/page.north", text, offset)
	if !ok || target.Name != "item" || len(target.Occurrences) != 2 {
		t.Fatalf("target = %#v, %v", target, ok)
	}
}
