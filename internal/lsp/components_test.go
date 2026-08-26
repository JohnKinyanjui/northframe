package lsp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectComponentsDiscoversPropsAndSlots(t *testing.T) {
	root := t.TempDir()
	route := filepath.Join(root, "web", "routes", "page.north")
	component := filepath.Join(root, "web", "components", "account", "user-card.north")
	if err := os.MkdirAll(filepath.Dir(route), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(component), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(route, []byte("<main></main>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(component, []byte(`---
interface Props {
Name string
Count int = 0
}
---
<script lang="ts">
function finish(): void { dispatch("complete", { count: props.Count }); }
</script>
<article><slot /><button on:click={finish}>Done</button></article>`), 0o644); err != nil {
		t.Fatal(err)
	}
	components := projectComponents(documentURI(route))
	if len(components) != 1 || components[0].Name != "AccountUserCard" || !components[0].HasSlot {
		t.Fatalf("components = %#v", components)
	}
	if len(components[0].Props) != 2 || components[0].Props[0] != "Name" || components[0].Props[1] != "Count" {
		t.Fatalf("props = %#v", components[0].Props)
	}
	if len(components[0].Events) != 1 || components[0].Events[0] != "complete" {
		t.Fatalf("events = %#v", components[0].Events)
	}
}
