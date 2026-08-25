package compiler

import (
	"strings"
	"testing"
)

func TestComponentTagsCompileTypedPropsAndSlots(t *testing.T) {
	generated, err := Compile("routes", "page", []byte(`<Card Title={Props.Title}><strong>{Props.Subtitle}</strong></Card>`))
	if err != nil {
		t.Fatal(err)
	}
	result := string(generated)
	for _, expected := range []string{`RenderCard(w, CardProps{`, `Title: props.Title`, `Content: func(w io.Writer) error`, `props.Subtitle`} {
		if !strings.Contains(result, expected) {
			t.Errorf("generated component call does not contain %q\n%s", expected, result)
		}
	}
}

func TestComponentTagsCompileBracedBooleanLiterals(t *testing.T) {
	generated, err := Compile("routes", "page", []byte(`<Sidebar Collapsed={false} />`))
	if err != nil {
		t.Fatal(err)
	}
	if result := string(generated); !strings.Contains(result, `Collapsed: false`) {
		t.Fatalf("generated component call does not contain a boolean literal\n%s", result)
	}
}

func TestComponentValidationRejectsMissingUnknownAndUnsupportedChildren(t *testing.T) {
	known := map[string]componentView{"Card": {Name: "Card", Props: []prop{{Name: "Title", Type: "string"}}}}
	for _, test := range []struct {
		source string
		want   string
	}{
		{source: `<Card />`, want: "required prop Title"},
		{source: `<Card Title="Hello" Colour="red" />`, want: "unknown prop Colour"},
		{source: `<Card Title="Hello">child</Card>`, want: "has no <slot />"},
		{source: `<Missing />`, want: "unknown component"},
	} {
		err := validateComponentReferences("Page", []byte(test.source), known)
		if err == nil || !strings.Contains(err.Error(), test.want) {
			t.Errorf("validate %q error = %v, want %q", test.source, err, test.want)
		}
	}
}
