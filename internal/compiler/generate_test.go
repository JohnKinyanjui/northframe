package compiler

import (
	"strings"
	"testing"
)

func TestCompileGeneratesNativeControlFlow(t *testing.T) {
	source := []byte(`<script context="props">
Title string
Visible bool
Items []string
</script>
<h1>{Props.Title}</h1>{if Props.Visible}<ul>{for item := range Props.Items}<li>{item}</li>{/for}</ul>{/if}`)

	generated, err := Compile("pages", "home", source)
	if err != nil {
		t.Fatalf("Compile returned an error: %v", err)
	}

	result := string(generated)
	for _, expected := range []string{
		"type HomeProps struct",
		"web.WriteEscaped(w, props.Title)",
		"web.Truthy(props.Visible)",
		"for _, item := range props.Items",
	} {
		if !strings.Contains(result, expected) {
			t.Errorf("generated renderer does not contain %q\n%s", expected, result)
		}
	}
}

func TestCompileSupportsInterfacePropsContract(t *testing.T) {
	source := []byte(`---
import models "example.test/app/viewmodels"

interface Props {
Title string
Items []models.Item
}
---
<h1>{Props.Title}</h1>{for item := range Props.Items}<p>{item.Name}</p>{/for}`)

	generated, err := Compile("components", "panel", source)
	if err != nil {
		t.Fatal(err)
	}
	result := string(generated)
	for _, expected := range []string{`models "example.test/app/viewmodels"`, "type PanelProps struct", "Title string", "Items []models.Item"} {
		if !strings.Contains(result, expected) {
			t.Errorf("generated renderer does not contain %q\n%s", expected, result)
		}
	}
}

func TestCompileRejectsDuplicatePropsContracts(t *testing.T) {
	_, err := Compile("components", "panel", []byte(`---
interface Props {
Title string
}
---
<script context="props">Title string</script><h1>{Props.Title}</h1>`))
	if err == nil || !strings.Contains(err.Error(), "declare props once") {
		t.Fatalf("error = %v", err)
	}
}

func TestCompileKeepsLegacyInterfacePropsCompatible(t *testing.T) {
	generated, err := Compile("components", "legacy", []byte(`interface props {
Title string
}
<h1>{Props.Title}</h1>`))
	if err != nil || !strings.Contains(string(generated), "type LegacyProps struct") {
		t.Fatalf("legacy interface props compatibility failed: %v\n%s", err, generated)
	}
}

func TestCompileRejectsJavaScriptExpressions(t *testing.T) {
	_, err := Compile("pages", "home", []byte(`<p>{items.map(x => x.name)}</p>`))
	if err == nil {
		t.Fatal("expected unsupported JavaScript expression to be rejected")
	}
}

func TestCompileSupportsNegatedIfConditions(t *testing.T) {
	source := []byte(`<script context="props">
Visible bool
</script>
{if !Props.Visible}<p>Hidden state</p>{/if}`)

	generated, err := Compile("pages", "home", source)
	if err != nil {
		t.Fatalf("Compile returned an error: %v", err)
	}
	if result := string(generated); !strings.Contains(result, "if !web.Truthy(props.Visible)") {
		t.Fatalf("generated renderer does not negate the condition\n%s", result)
	}
}

func TestCompileSupportsGoExpressionsInServerBlocks(t *testing.T) {
	source := []byte(`---
interface Props {
Sections [][]string
}
---
{for section := range Props.Sections}
  {if len(section) > 0}<p>{len(section)}</p>{/if}
{/for}`)

	generated, err := Compile("pages", "sections", source)
	if err != nil {
		t.Fatalf("Compile returned an error: %v", err)
	}
	result := string(generated)
	for _, expected := range []string{"for _, section := range props.Sections", "if len(section) > 0", "web.WriteEscaped(w, len(section))"} {
		if !strings.Contains(result, expected) {
			t.Errorf("generated renderer does not contain %q\n%s", expected, result)
		}
	}
}

func TestCompileExplainsRetiredSvelteServerSyntax(t *testing.T) {
	_, err := Compile("pages", "legacy", []byte(`{#each $Items as item}{/each}`))
	if err == nil || !strings.Contains(err.Error(), "Svelte-style server blocks are no longer supported") {
		t.Fatalf("error = %v", err)
	}
}

func TestCompileKeepsStyleAtItsDocumentPosition(t *testing.T) {
	generated, err := Compile("pages", "document", []byte(`<!doctype html><head><style>h1 { color: red; }</style></head><body>ok</body>`))
	if err != nil {
		t.Fatalf("Compile returned an error: %v", err)
	}

	result := string(generated)
	doctype := strings.Index(result, `<!doctype html><head>`)
	style := strings.Index(result, `<style>h1 { color: red; }</style>`)
	body := strings.Index(result, `</head><body>ok</body>`)
	if doctype < 0 || style < doctype || body < style {
		t.Fatalf("style moved outside its document position:\n%s", result)
	}
}

func TestCompileSupportsImportedViewModels(t *testing.T) {
	source := []byte(`<script context="props">
import models "example.test/app/viewmodels"
Users []models.User
</script>
{for user := range Props.Users}<p>{user.Name}</p>{/for}`)
	generated, err := Compile("pages", "users", source)
	if err != nil {
		t.Fatalf("Compile returned an error: %v", err)
	}
	result := string(generated)
	for _, expected := range []string{`models "example.test/app/viewmodels"`, `Users []models.User`, `props.Users`, `user.Name`} {
		if !strings.Contains(result, expected) {
			t.Errorf("generated renderer does not contain %q\n%s", expected, result)
		}
	}
}

func TestCompileRouteUsesSidecarPropsAndExplicitLayoutContent(t *testing.T) {
	generated, err := CompileRoute("routesgen", "RootLayout", []byte(`<html><body><h1>{Props.Title}</h1><slot /></body></html>`), "routeRoot", "example.test/app/routes", "layout")
	if err != nil {
		t.Fatal(err)
	}
	result := string(generated)
	for _, expected := range []string{`props routeRoot.LayoutProps`, `content web.Fragment`, `content(w)`, `props.Title`} {
		if !strings.Contains(result, expected) {
			t.Errorf("generated route renderer does not contain %q\n%s", expected, result)
		}
	}
}
