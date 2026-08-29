package lsp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPropsImportCompletionDiscoversModulePackages(t *testing.T) {
	root := t.TempDir()
	writeImportTestFile(t, filepath.Join(root, "go.mod"), "module example.test/store\n\ngo 1.27\n")
	writeImportTestFile(t, filepath.Join(root, "internal", "viewmodels", "models.go"), "package viewmodels\ntype Product struct{}\n")
	component := filepath.Join(root, "web", "components", "card.north")
	text := `---
import view

interface Props {
}
---`
	writeImportTestFile(t, component, text)
	offset := len(`---
import view`)
	items := propsImportCompletionItems(documentURI(component), text, offset)
	found := false
	for _, item := range items {
		if item["label"] == "example.test/store/internal/viewmodels" {
			found = true
			edit := item["textEdit"].(map[string]any)
			if edit["newText"] != `viewmodels "example.test/store/internal/viewmodels"` {
				t.Fatalf("newText = %q", edit["newText"])
			}
		}
	}
	if !found {
		t.Fatalf("module import missing from %#v", items)
	}
}

func TestImportedGoTypesCompleteAndNavigate(t *testing.T) {
	root := t.TempDir()
	writeImportTestFile(t, filepath.Join(root, "go.mod"), "module example.test/store\n\ngo 1.27\n")
	modelsPath := filepath.Join(root, "internal", "viewmodels", "models.go")
	writeImportTestFile(t, modelsPath, `package viewmodels

// NavigationItem is shared navigation data.
type NavigationItem interface {
	Label() string
}

type Product struct{}
type privateModel struct{}
`)
	component := filepath.Join(root, "web", "components", "card.north")
	text := `---
import viewmodels "example.test/store/internal/viewmodels"

interface Props {
  Item viewmodels.Nav
}
---`
	writeImportTestFile(t, component, text)
	offset := strings.Index(text, "viewmodels.Nav") + len("viewmodels.Nav")
	items := goTypeCompletionItems(documentURI(component), text, offset)
	if len(items) != 1 {
		t.Fatalf("completion items = %#v", items)
	}
	if items[0]["label"] != "NavigationItem" {
		t.Fatalf("completion labels = %#v", items)
	}

	resolvedText := strings.Replace(text, "viewmodels.Nav", "viewmodels.NavigationItem", 1)
	resolvedOffset := strings.Index(resolvedText, "NavigationItem") + 3
	symbol, ok := importedGoTypeAt(documentURI(component), resolvedText, resolvedOffset)
	if !ok || symbol.Name != "NavigationItem" || symbol.URI != documentURI(modelsPath) {
		t.Fatalf("resolved symbol = %#v, %v", symbol, ok)
	}
	if symbol.Kind != 8 || !strings.Contains(symbol.Documentation, "shared navigation data") {
		t.Fatalf("symbol metadata = %#v", symbol)
	}
}

func TestGoTypeCompletionAutoImportsProjectPackage(t *testing.T) {
	root := t.TempDir()
	writeImportTestFile(t, filepath.Join(root, "go.mod"), "module example.test/store\n\ngo 1.27\n")
	writeImportTestFile(t, filepath.Join(root, "internal", "viewmodels", "models.go"), "package viewmodels\ntype NavItem struct { Label string }\n")
	component := filepath.Join(root, "web", "components", "sidebar.north")
	text := `---
interface Props {
  Item viewmodels.Nav
}
---`
	writeImportTestFile(t, component, text)
	offset := strings.Index(text, "viewmodels.Nav") + len("viewmodels.Nav")
	items := goTypeCompletionItems(documentURI(component), text, offset)
	if len(items) != 1 || items[0]["label"] != "NavItem" {
		t.Fatalf("completion items = %#v", items)
	}
	edits, ok := items[0]["additionalTextEdits"].([]map[string]any)
	if !ok || len(edits) != 1 || edits[0]["newText"] != "import viewmodels \"example.test/store/internal/viewmodels\"\n\n" {
		t.Fatalf("auto import edits = %#v", items[0]["additionalTextEdits"])
	}
}

func TestUnqualifiedGoTypeCompletionAddsQualifierAndImport(t *testing.T) {
	root := t.TempDir()
	writeImportTestFile(t, filepath.Join(root, "go.mod"), "module example.test/store\n\ngo 1.27\n")
	writeImportTestFile(t, filepath.Join(root, "internal", "viewmodels", "models.go"), "package viewmodels\ntype NavItem struct { Label string }\n")
	component := filepath.Join(root, "web", "components", "sidebar.north")
	text := `---
interface Props {
  Item Nav
}
---`
	offset := strings.Index(text, "Nav") + len("Nav")
	items := goTypeCompletionItems(documentURI(component), text, offset)
	if len(items) != 1 {
		t.Fatalf("completion items = %#v", items)
	}
	edit := items[0]["textEdit"].(map[string]any)
	if edit["newText"] != "viewmodels.NavItem" {
		t.Fatalf("type edit = %#v", edit)
	}
}

func TestExternalGoTypeCompletionUsesGoModulePackages(t *testing.T) {
	root := t.TempDir()
	writeImportTestFile(t, filepath.Join(root, "go.mod"), `module example.test/store

go 1.27

require example.test/uuid v0.0.0

replace example.test/uuid => ./third_party/uuid
`)
	writeImportTestFile(t, filepath.Join(root, "third_party", "uuid", "go.mod"), "module example.test/uuid\n\ngo 1.27\n")
	writeImportTestFile(t, filepath.Join(root, "third_party", "uuid", "uuid.go"), "package uuid\ntype UUID [16]byte\n")
	component := filepath.Join(root, "web", "components", "card.north")
	text := `---
interface Props {
  ID uuid.UU
}
---`
	writeImportTestFile(t, component, text)
	offset := strings.Index(text, "uuid.UU") + len("uuid.UU")
	items := goTypeCompletionItems(documentURI(component), text, offset)
	var external map[string]any
	for _, item := range items {
		if strings.Contains(item["detail"].(string), "example.test/uuid") {
			external = item
			break
		}
	}
	if external == nil || external["label"] != "UUID" {
		t.Fatalf("completion items = %#v", items)
	}
	edits, ok := external["additionalTextEdits"].([]map[string]any)
	if !ok || len(edits) != 1 || edits[0]["newText"] != "import uuid \"example.test/uuid\"\n\n" {
		t.Fatalf("auto import edits = %#v", external["additionalTextEdits"])
	}
	resolved := strings.Replace(text, "interface Props {", "import uuid \"example.test/uuid\"\n\ninterface Props {", 1)
	resolved = strings.Replace(resolved, "uuid.UU", "uuid.UUID", 1)
	symbol, found := importedGoTypeAt(documentURI(component), resolved, strings.Index(resolved, "uuid.UUID")+len("uuid.UUID")-1)
	if !found || symbol.Name != "UUID" || !strings.HasSuffix(documentPath(symbol.URI), filepath.Join("third_party", "uuid", "uuid.go")) {
		t.Fatalf("resolved symbol = %#v, %v", symbol, found)
	}
}

func TestOrganizePropsImportsAddsMissingPackagesAndSortsImports(t *testing.T) {
	root := t.TempDir()
	writeImportTestFile(t, filepath.Join(root, "go.mod"), `module example.test/store

go 1.27

require example.test/uuid v0.0.0

replace example.test/uuid => ./third_party/uuid
`)
	writeImportTestFile(t, filepath.Join(root, "third_party", "uuid", "go.mod"), "module example.test/uuid\n\ngo 1.27\n")
	writeImportTestFile(t, filepath.Join(root, "third_party", "uuid", "uuid.go"), "package uuid\ntype UUID [16]byte\n")
	writeImportTestFile(t, filepath.Join(root, "internal", "viewmodels", "models.go"), "package viewmodels\ntype Product struct{}\n")
	component := filepath.Join(root, "web", "components", "card.north")
	text := `---
import viewmodels "example.test/store/internal/viewmodels"

interface Props {
  ID uuid.UUID
  Item viewmodels.Product
}
---
<article>${Props.ID}</article>`
	writeImportTestFile(t, component, text)
	imports := projectImports(documentURI(component))
	if !containsProjectImport(imports, "uuid", "example.test/uuid") {
		t.Fatalf("project imports = %#v", imports)
	}

	got := organizePropsImports(documentURI(component), text)
	want := `---
import viewmodels "example.test/store/internal/viewmodels"
import uuid "example.test/uuid"

interface Props {
  ID uuid.UUID
  Item viewmodels.Product
}
---
<article>${Props.ID}</article>`
	if got != want {
		t.Fatalf("organizePropsImports() =\n%s\nwant:\n%s", got, want)
	}
	if again := organizePropsImports(documentURI(component), got); again != got {
		t.Fatalf("organizePropsImports is not idempotent:\n%s", again)
	}
}

func containsProjectImport(imports []projectImport, alias, path string) bool {
	for _, imported := range imports {
		if imported.Alias == alias && imported.Path == path {
			return true
		}
	}
	return false
}

func writeImportTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
