package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"northframe.dev/northframe/internal/dependencies"
)

func TestCompileClientComponentTransformsTypeScriptState(t *testing.T) {
	source := []byte(`<script lang="ts">
let open: boolean = false;
let name: string = "Relay";
function toggle() { open = !open; }
</script>
<button on:click={toggle} aria-expanded={#open}>Add product</button>
<section show={#open}><input bind:value={#name}><strong>{#name}</strong></section>
<p>{Props.ServerTitle}</p>`)
	markup, module, err := compileClientComponent("InventoryPage", source)
	if err != nil {
		t.Fatal(err)
	}
	if module == nil || !strings.HasPrefix(module.Path, "inventory-page-") || !strings.HasSuffix(module.Path, ".js") {
		t.Fatalf("module = %#v", module)
	}
	for _, expected := range []string{"data-north-event-inventory-page", "data-north-bind-inventory-page", "{Props.ServerTitle}", `/_northframe/components/inventory-page-`} {
		if !strings.Contains(string(markup), expected) {
			t.Errorf("markup does not contain %q\n%s", expected, markup)
		}
	}
	javascript := string(module.Source)
	for _, expected := range []string{"let open = false", "mountComponent", "open = !open", `kind: "model"`} {
		if !strings.Contains(javascript, expected) {
			t.Errorf("module does not contain %q\n%s", expected, javascript)
		}
	}
	if strings.Contains(javascript, ": boolean") || strings.Contains(javascript, ": string") {
		t.Errorf("TypeScript types were not removed\n%s", javascript)
	}
}

func TestCompileClientComponentPreservesInterfaceProps(t *testing.T) {
	source := []byte(`---
interface Props {
Title string
}
---
<script lang="ts">let open: boolean = false;</script>
<button on:click={() => open = !open}>{Props.Title}</button>`)
	markup, module, err := compileClientComponent("Panel", source)
	if err != nil {
		t.Fatal(err)
	}
	if module == nil || !strings.HasPrefix(string(markup), "---\ninterface Props") || !strings.Contains(string(markup), "Title string") {
		t.Fatalf("Props frontmatter contract was not preserved\n%s", markup)
	}
}

func TestValidateAllowsSpacedPropsFrontmatter(t *testing.T) {
	source := []byte(`---

import viewmodels "example/internal/viewmodels"

interface Props {
  Item viewmodels.NavItem
}

---

<p>{Props.Item.Label}</p>`)
	if err := Validate(source); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	prepared, _, err := compileClientComponent("SidebarGroup", source)
	if err != nil {
		t.Fatalf("compileClientComponent() error = %v", err)
	}
	generated, err := Compile("routes", "SidebarGroup", prepared)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if !strings.Contains(string(generated), `viewmodels "example/internal/viewmodels"`) {
		t.Fatalf("generated renderer omitted props import:\n%s", generated)
	}
}

func TestCompileClientComponentPreservesNegatedServerCondition(t *testing.T) {
	markup, _, err := compileClientComponent("Navigation", []byte(`{if !Props.Item.Active}<a>Inactive</a>{/if}`))
	if err != nil {
		t.Fatal(err)
	}
	if result := string(markup); result != `{if !Props.Item.Active}<a>Inactive</a>{/if}` {
		t.Fatalf("markup = %q", result)
	}
}

func TestCompileClientComponentRequiresScriptForClientState(t *testing.T) {
	_, _, err := compileClientComponent("Page", []byte(`<p>{#count}</p>`))
	if err == nil || !strings.Contains(err.Error(), `<script lang="ts">`) {
		t.Fatalf("error = %v", err)
	}
}

func TestCompileClientComponentRejectsInvalidTypeScript(t *testing.T) {
	_, _, err := compileClientComponent("Page", []byte(`<script lang="ts">let count: = 1;</script><p>{#count}</p>`))
	if err == nil || !strings.Contains(err.Error(), "TypeScript") {
		t.Fatalf("error = %v", err)
	}
}

func TestCompileClientComponentRejectsTypeMismatch(t *testing.T) {
	_, _, err := compileClientComponent("Page", []byte(`<script lang="ts">let count: number = "wrong";</script><p>{#count}</p>`))
	if err == nil || !strings.Contains(err.Error(), "declared number but initialized with string") {
		t.Fatalf("error = %v", err)
	}
}

func TestCompileClientComponentRejectsUnknownState(t *testing.T) {
	_, _, err := compileClientComponent("Page", []byte(`<script lang="ts">let count: number = 1;</script><p>{#missing}</p>`))
	if err == nil || !strings.Contains(err.Error(), "browser state missing is not declared") {
		t.Fatalf("error = %v", err)
	}
}

func TestCompileClientComponentChecksGeneratedGoProps(t *testing.T) {
	_, _, err := compileClientComponentWithOptions("Page", []byte(`<script lang="ts">let count: number = props.Query;</script><p>{#count}</p>`), clientCompileOptions{
		Contract: `type PageProps = { readonly Query: string };`, PropsType: "PageProps", Props: []prop{{Name: "Query", Type: "string"}},
	})
	if err == nil || !strings.Contains(err.Error(), "initialized with string") {
		t.Fatalf("error = %v", err)
	}
}

func TestCompileClientComponentProvidesTypedActionEvents(t *testing.T) {
	source := []byte(`<script lang="ts">
let message: string = "";
function invalid(event: NorthframeActionEvent): void { message = event.detail.result.message || "Invalid"; }
</script><form on:northframe-invalid={invalid}><p>{#message}</p></form>`)
	_, module, err := compileClientComponent("TypedActionPage", source)
	if err != nil {
		t.Fatal(err)
	}
	if module == nil || !strings.Contains(string(module.Source), "event.detail.result.message") {
		t.Fatalf("compiled module does not contain action handler\n%s", module.Source)
	}
}

func TestExtractClientImportsSupportsSideEffectImports(t *testing.T) {
	imports, body := extractClientImports("import \"iconify-icon\";\n\nlet open: boolean = false;")
	if imports != `import "iconify-icon";` {
		t.Fatalf("imports = %q", imports)
	}
	if body != "let open: boolean = false;" {
		t.Fatalf("body = %q", body)
	}
}

func TestCompileClientComponentBundlesManagedAndLocalImports(t *testing.T) {
	root := t.TempDir()
	writeClientTestFile(t, filepath.Join(root, "client", "math.ts"), `export function double(value: number): number { return value * 2; }`)
	writeClientTestFile(t, filepath.Join(root, ".northframe", "modules", "node_modules", "tiny", "package.json"), `{"name":"tiny","version":"1.0.0","module":"index.js"}`)
	writeClientTestFile(t, filepath.Join(root, ".northframe", "modules", "node_modules", "tiny", "index.js"), `export const answer = 21;`)
	sourcePath := filepath.Join(root, "routes", "page.north")
	writeClientTestFile(t, sourcePath, "")
	source := []byte(`<script lang="ts">
import { answer } from "tiny";
import { double } from "$client/math";
let count: number = double(answer);
function increment(): void { count += 1; }
</script><button on:click={increment}>{#count}</button>`)
	_, module, err := compileClientComponentWithOptions("ImportedPage", source, clientCompileOptions{
		SourcePath: sourcePath,
		Project: dependencies.Project{
			Root: root, ClientSource: filepath.Join(root, "client"),
			NodeModules:  filepath.Join(root, ".northframe", "modules", "node_modules"),
			Dependencies: map[string]string{"tiny": "1.0.0"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	javascript := string(module.Source)
	for _, expected := range []string{"value * 2", "var answer = 21", "double(answer)"} {
		if !strings.Contains(javascript, expected) {
			t.Errorf("bundle does not contain %q\n%s", expected, javascript)
		}
	}
	if strings.Contains(javascript, `from "tiny"`) || strings.Contains(javascript, "$client") {
		t.Errorf("imports were not bundled\n%s", javascript)
	}
}

func TestCompileClientComponentGuidesUndeclaredPackageImport(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "routes", "page.north")
	writeClientTestFile(t, sourcePath, "")
	_, _, err := compileClientComponentWithOptions("Page", []byte(`<script lang="ts">import thing from "thing"; let open: boolean = Boolean(thing);</script><p>{#open}</p>`), clientCompileOptions{
		SourcePath: sourcePath,
		Project:    dependencies.Project{Root: root, ClientSource: filepath.Join(root, "client"), NodeModules: filepath.Join(root, ".northframe", "modules", "node_modules"), Dependencies: map[string]string{}},
	})
	if err == nil || !strings.Contains(err.Error(), "north add thing") {
		t.Fatalf("error = %v", err)
	}
}

func TestCompileScopedClientComponentHoistsImports(t *testing.T) {
	root := t.TempDir()
	writeClientTestFile(t, filepath.Join(root, "client", "initial.ts"), `export const initial: number = 3;`)
	sourcePath := filepath.Join(root, "components", "counter.north")
	writeClientTestFile(t, sourcePath, "")
	source := []byte(`<script lang="ts">
import {
  initial,
} from "$client/initial";
let count: number = initial;
</script><p>{#count}</p>`)
	_, module, err := compileClientComponentWithOptions("Counter", source, clientCompileOptions{
		Scoped: true, SourcePath: sourcePath,
		Project: dependencies.Project{Root: root, ClientSource: filepath.Join(root, "client"), NodeModules: filepath.Join(root, ".northframe", "modules", "node_modules"), Dependencies: map[string]string{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if module == nil || !strings.Contains(string(module.Source), "var initial = 3") {
		t.Fatalf("scoped import was not bundled\n%s", module.Source)
	}
}

func writeClientTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
