package lsp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitializeAdvertisesLanguageFeatures(t *testing.T) {
	input := frame(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`) +
		frame(`{"jsonrpc":"2.0","method":"exit"}`)
	var output bytes.Buffer
	if err := Run(strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	message := decodeFirstFrame(t, output.String())
	result := message["result"].(map[string]any)
	capabilities := result["capabilities"].(map[string]any)
	if capabilities["definitionProvider"] != true || capabilities["documentFormattingProvider"] != true || capabilities["renameProvider"] == nil {
		t.Fatalf("unexpected capabilities: %#v", capabilities)
	}
}

func TestDiagnosticsUseCompilerParser(t *testing.T) {
	diagnostics := validateDocument("file:///tmp/page.north", "<h1>{if Props.Ready}broken")
	if len(diagnostics) == 0 || !strings.Contains(diagnostics[0].Message, "closed") {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
	if diagnostics[0].Range.Start.Character != 4 {
		t.Fatalf("diagnostic starts at %#v, want character 4", diagnostics[0].Range.Start)
	}
}

func TestDiagnosticsWarnForUnsupportedUtilityClass(t *testing.T) {
	diagnostics := validateDocument("file:///tmp/Card.north", `<div class="flex definitely-not-tailwind"></div>`)
	found := false
	for _, current := range diagnostics {
		if strings.Contains(current.Message, `unsupported utility class "definitely-not-tailwind"`) {
			found = true
			if current.Severity != 2 {
				t.Fatalf("severity = %d, want warning", current.Severity)
			}
		}
	}
	if !found {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
}

func TestDiagnosticsAllowSpacedPropsFrontmatter(t *testing.T) {
	diagnostics := validateDocument("file:///tmp/component.north", `---

  import viewmodels "example/internal/viewmodels"

interface Props {
  Item viewmodels.NavItem
}
---

<p>{Props.Item.Label}</p>`)
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
}

func TestDiagnosticsAllowMultilineServerDirectives(t *testing.T) {
	diagnostics := validateDocument("file:///tmp/Card.north", `---
interface Props {
  Enabled bool
  Items []string
}
---
{if
  !Props.Enabled}<span>Disabled</span>{/if}
{for
  item := range Props.Items}<span>{item}</span>{/for}`)
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
}

func TestDiagnosticsDoNotTreatGoImportPathAsQualifiedType(t *testing.T) {
	diagnostics := validateDocument("file:///tmp/component.north", `---
import uuid "github.com/google/uuid"

interface Props {
  SetupRequired bool
  ID uuid.UUID
}
---`)
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
}

func TestDiagnosticsExplainMissingProjectTypeImport(t *testing.T) {
	root := t.TempDir()
	writeImportTestFile(t, filepath.Join(root, "go.mod"), "module example.test/store\n\ngo 1.27\n")
	writeImportTestFile(t, filepath.Join(root, "internal", "viewmodels", "models.go"), "package viewmodels\ntype NavItem struct{}\n")
	component := filepath.Join(root, "components", "sidebar.north")
	text := `---
interface Props {
  Item viewmodels.NavItem
}
---`
	diagnostics := validateDocument(documentURI(component), text)
	if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Message, "add `import viewmodels") {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
}

func TestSidecarPropsProvideTypedLocations(t *testing.T) {
	directory := t.TempDir()
	templatePath := filepath.Join(directory, "page.north")
	sidecar := `package routes
type PageProps struct {
	Title string
	Count int
}
`
	if err := os.WriteFile(templatePath+".go", []byte(sidecar), 0o644); err != nil {
		t.Fatal(err)
	}
	fields := sidecarProps(documentURI(templatePath))
	if len(fields) != 2 || fields[0].Name != "Title" || fields[0].Type != "string" {
		t.Fatalf("sidecarProps() = %#v", fields)
	}
	if !strings.HasSuffix(fields[0].URI, "page.north.go") {
		t.Fatalf("definition URI = %q", fields[0].URI)
	}
}

func TestTemplatePropsProvideGeneratedTypedLocations(t *testing.T) {
	directory := t.TempDir()
	templatePath := filepath.Join(directory, "page.north")
	template := `---
interface Props {
Title string
Count int
}
---
<h1>{Props.Title}</h1>`
	if err := os.WriteFile(templatePath, []byte(template), 0o644); err != nil {
		t.Fatal(err)
	}
	fields := sidecarProps(documentURI(templatePath))
	if len(fields) != 2 || fields[0].Name != "Title" || fields[1].Type != "int" {
		t.Fatalf("sidecarProps() = %#v", fields)
	}
	if !strings.HasSuffix(fields[0].URI, "page.north") {
		t.Fatalf("definition URI = %q", fields[0].URI)
	}
}

func TestComponentPropsProvideTypedLocations(t *testing.T) {
	directory := t.TempDir()
	templatePath := filepath.Join(directory, "Sidebar.north")
	template := `---
interface Props {
Sections []string
}
---
{for section := range Props.Sections}<p>{section}</p>{/for}`
	if err := os.WriteFile(templatePath, []byte(template), 0o644); err != nil {
		t.Fatal(err)
	}
	fields := sidecarProps(documentURI(templatePath))
	if len(fields) != 1 || fields[0].Name != "Sections" || fields[0].Type != "[]string" {
		t.Fatalf("component props = %#v", fields)
	}
}

func TestTemplateResolverFollowsLoopVariablesAndGoStructFields(t *testing.T) {
	root := t.TempDir()
	writeImportTestFile(t, filepath.Join(root, "go.mod"), "module example.test/store\n\ngo 1.27\n")
	modelsPath := filepath.Join(root, "internal", "viewmodels", "models.go")
	writeImportTestFile(t, modelsPath, `package viewmodels
type NavSection struct {
	Title string
	Items []NavItem
}
type NavItem struct { Label string }
`)
	templatePath := filepath.Join(root, "web", "components", "sidebar.north")
	template := `---
import viewmodels "example.test/store/internal/viewmodels"

interface Props {
  Sections []viewmodels.NavSection
}
---
{for section := range Props.Sections}
  <h2>{section.Title}</h2>
  {for item := range section.Items}<p>{item.Label}</p>{/for}
{/for}`
	writeImportTestFile(t, templatePath, template)
	uri := documentURI(templatePath)
	sectionsOffset := strings.Index(template, "Props.Sections") + len("Props.Sections") - 2
	sections, ok := templateValueAt(uri, template, sectionsOffset)
	if !ok || sections.Type != "[]viewmodels.NavSection" || !sections.Prop {
		t.Fatalf("Sections = %#v, %v", sections, ok)
	}
	titleOffset := strings.Index(template, "section.Title") + len("section.Title") - 2
	title, ok := templateValueAt(uri, template, titleOffset)
	if !ok || title.Type != "string" || title.Owner != "viewmodels.NavSection" || title.URI != documentURI(modelsPath) {
		t.Fatalf("section.Title = %#v, %v", title, ok)
	}
	labelOffset := strings.Index(template, "item.Label") + len("item.Label") - 2
	label, ok := templateValueAt(uri, template, labelOffset)
	if !ok || label.Type != "string" || label.Owner != "viewmodels.NavItem" {
		t.Fatalf("item.Label = %#v, %v", label, ok)
	}
	items := templateFieldCompletionItems(uri, template, strings.Index(template, "section.Title")+len("section."))
	if len(items) != 2 || items[0]["label"] != "Title" || items[1]["label"] != "Items" {
		t.Fatalf("section completions = %#v", items)
	}
}

func TestHTMLDirectiveDiagnosticsRequireSafeHTML(t *testing.T) {
	root := t.TempDir()
	writeImportTestFile(t, filepath.Join(root, "go.mod"), "module example.test/store\n\ngo 1.27\n")
	writeImportTestFile(t, filepath.Join(root, "internal", "viewmodels", "models.go"), `package viewmodels
import "northframe.dev/northframe/pkg/web"
type Article struct {
	Content string
	ContentHTML web.SafeHTML
}
`)
	templatePath := filepath.Join(root, "web", "components", "article.north")
	template := `---
import viewmodels "example.test/store/internal/viewmodels"
interface Props {
  Articles []viewmodels.Article
}
---
{for item := range Props.Articles}
  <div>{html item.ContentHTML}</div>
  <div>{html item.Content}</div>
{/for}`
	writeImportTestFile(t, templatePath, template)
	diagnostics := htmlTypeDiagnostics(documentURI(templatePath), template)
	if len(diagnostics) != 1 || !strings.Contains(diagnostics[0].Message, "requires web.SafeHTML") || !strings.Contains(diagnostics[0].Message, "type string") {
		t.Fatalf("html diagnostics = %#v", diagnostics)
	}
}

func TestFormatDocument(t *testing.T) {
	got := formatDocument("<main>  \n<slot/>\n\n")
	if got != "<main>\n<slot />\n" {
		t.Fatalf("formatDocument() = %q", got)
	}
}

func TestHoverHelpExplainsNorthframeSyntaxAtCursor(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		token  string
		wanted string
	}{
		{"props contract", "---\ninterface Props {\nTitle string\n}\n---", "Props", "Typed server Props"},
		{"typescript", `<script lang="ts">`, "ts", "Browser TypeScript"},
		{"server loop", `{for item := range Props.Items}`, "for", "Go server loop"},
		{"sanitized html", `{html Props.ContentHTML}`, "html", "Sanitized server HTML"},
		{"client state", `<p>{#name}</p>`, "name", "Client expression"},
		{"pending form", `<span nf-loading hidden>Saving</span>`, "nf-loading", "Pending state"},
		{"component dispatch", `function done() { dispatch("complete", { id: 1 }); }`, "dispatch", "Component event dispatch"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			offset := strings.Index(test.text, test.token)
			help, ok := hoverHelpAt(test.text, offset)
			if !ok || !strings.Contains(help, test.wanted) {
				t.Fatalf("hoverHelpAt() = %q, %v; want %q", help, ok, test.wanted)
			}
		})
	}
}

func frame(payload string) string {
	return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(payload), payload)
}

func decodeFirstFrame(t *testing.T, raw string) map[string]any {
	t.Helper()
	_, payload, found := strings.Cut(raw, "\r\n\r\n")
	if !found {
		t.Fatalf("invalid frame %q", raw)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded
}
