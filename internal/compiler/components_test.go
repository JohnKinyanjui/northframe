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

func TestComponentTagsSupportDollarServerProps(t *testing.T) {
	generated, err := Compile("routes", "page", []byte(`<Card Title=${Props.Title} Count=${Props.Count} />`))
	if err != nil {
		t.Fatal(err)
	}
	result := string(generated)
	for _, expected := range []string{`Title: props.Title`, `Count: props.Count`} {
		if !strings.Contains(result, expected) {
			t.Errorf("generated component call does not contain %q\n%s", expected, result)
		}
	}
}

func TestHTMLDirectiveCompilesToTypedSafeWriter(t *testing.T) {
	generated, err := Compile("routes", "page", []byte(`<article>{html Props.Content}</article>`))
	if err != nil {
		t.Fatal(err)
	}
	result := string(generated)
	if !strings.Contains(result, `web.WriteHTML(w, props.Content)`) {
		t.Fatalf("generated HTML directive does not use the typed safe writer\n%s", result)
	}
	if strings.Contains(result, `web.WriteEscaped(w, props.Content)`) {
		t.Fatalf("generated HTML directive was escaped as ordinary text\n%s", result)
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

func TestComponentDefaultsAreOptionalAndInjectedAtCallSite(t *testing.T) {
	definition := componentView{Name: "Badge", Props: []prop{
		{Name: "Label", Type: "string"},
		{Name: "Tone", Type: "string", Default: `"neutral"`, HasDefault: true},
		{Name: "Dismissible", Type: "bool", Default: "true", HasDefault: true},
	}}
	known := map[string]componentView{"Badge": definition}
	if err := validateComponentReferences("Page", []byte(`<Badge Label="Ready" />`), known); err != nil {
		t.Fatalf("defaulted props should be optional: %v", err)
	}
	generated, err := compileComponent("routes", "page", []byte(`<Badge Label="Ready" />`), known)
	if err != nil {
		t.Fatal(err)
	}
	result := string(generated)
	for _, expected := range []string{`Label:`, `"Ready"`, `northDefaultBadgeTone()`, `northDefaultBadgeDismissible()`} {
		if !strings.Contains(result, expected) {
			t.Errorf("generated defaults do not contain %q\n%s", expected, result)
		}
	}

	explicit, err := compileComponent("routes", "page", []byte(`<Badge Label="Ready" Tone="danger" Dismissible={false} />`), known)
	if err != nil {
		t.Fatal(err)
	}
	result = string(explicit)
	if strings.Count(result, `Tone:`) != 1 || !strings.Contains(result, `"danger"`) || !strings.Contains(result, `Dismissible: false`) {
		t.Fatalf("explicit zero/default overrides were not preserved\n%s", result)
	}
}

func TestComponentDefaultsResolveInsideTheComponentImportScope(t *testing.T) {
	definitionSource := []byte(`---
import time "time"

interface Props {
    At time.Time = time.Time{}
}
---
<time>{Props.At}</time>`)
	props, imports, err := componentProps(definitionSource)
	if err != nil {
		t.Fatal(err)
	}
	known := map[string]componentView{"Clock": {Name: "Clock", Props: props, Imports: imports}}
	component, err := compileComponent("routes", "Clock", definitionSource, known)
	if err != nil {
		t.Fatal(err)
	}
	page, err := compileComponent("routes", "Page", []byte(`<Clock />`), known)
	if err != nil {
		t.Fatal(err)
	}
	if result := string(component); !strings.Contains(result, `func northDefaultClockAt() time.Time`) || !strings.Contains(result, `return time.Time{}`) {
		t.Fatalf("component default is not evaluated in its import scope\n%s", result)
	}
	if result := string(page); !strings.Contains(result, `At: northDefaultClockAt()`) || strings.Contains(result, `time.Time{}`) {
		t.Fatalf("caller leaked the component default expression\n%s", result)
	}
}

func TestParsePropsAcceptsGoDefaults(t *testing.T) {
	props, _, err := parseProps("Title string = \"Untitled\"\nOpen bool = false")
	if err != nil {
		t.Fatal(err)
	}
	if len(props) != 2 || !props[0].HasDefault || props[0].Default != `"Untitled"` || props[1].Default != "false" {
		t.Fatalf("unexpected props: %#v", props)
	}
}

func TestNamedSlotsCompileAsIndependentGoFragments(t *testing.T) {
	definitionSource := []byte(`<section><header><slot name="actions" /></header><main><slot /></main><footer><slot name="footer" /></footer></section>`)
	parsed, err := parseComponent("Panel", string(definitionSource))
	if err != nil {
		t.Fatal(err)
	}
	definition := componentView{Name: "Panel", Source: definitionSource, HasSlot: true, Slots: componentSlotNames(string(definitionSource))}
	known := map[string]componentView{"Panel": definition}
	call := []byte(`<Panel><p>Body</p><Fragment Name="actions"><button>Save</button></Fragment><Fragment Name="footer"><small>Ready</small></Fragment></Panel>`)
	if err := validateComponentReferences("Page", call, known); err != nil {
		t.Fatal(err)
	}
	component, err := compileComponent("routes", "Panel", definitionSource, known)
	if err != nil {
		t.Fatal(err)
	}
	page, err := compileComponent("routes", "Page", call, known)
	if err != nil {
		t.Fatal(err)
	}
	combined := string(component) + string(page)
	compact := strings.Join(strings.Fields(combined), " ")
	for _, expected := range []string{`SlotActions web.Fragment`, `SlotFooter web.Fragment`, `props.SlotActions`, `SlotActions: func`, `<button>Save</button>`, `Content: func`, `<p>Body</p>`} {
		if !strings.Contains(compact, expected) {
			t.Errorf("named slot output lacks %q\n%s", expected, combined)
		}
	}
	_ = parsed
}

func TestNamedSlotValidationRejectsUnknownAndDuplicateFragments(t *testing.T) {
	known := map[string]componentView{"Panel": {Name: "Panel", HasSlot: true, Slots: []string{"", "actions"}}}
	for _, test := range []struct{ source, want string }{
		{`<Panel><Fragment Name="missing">x</Fragment></Panel>`, `unknown slot "missing"`},
		{`<Panel><Fragment Name="actions">x</Fragment><Fragment Name="actions">y</Fragment></Panel>`, `more than once`},
		{`<Fragment Name="actions">x</Fragment>`, `outside a component`},
	} {
		err := validateComponentReferences("Page", []byte(test.source), known)
		if err == nil || !strings.Contains(err.Error(), test.want) {
			t.Errorf("validate %q = %v, want %q", test.source, err, test.want)
		}
	}
}

func TestComponentEventsCompileToBubblingParentListeners(t *testing.T) {
	parentSource := []byte(`<script lang="ts">
let closed: boolean = false;
function handleClose(event: CustomEvent<{ reason: string }>): void { closed = event.detail.reason === "done"; }
</script>
<Dialog on:close={handleClose} />`)
	markup, parentModule, err := compileClientComponentWithOptions("Page", parentSource, clientCompileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if parentModule == nil || !strings.Contains(string(markup), "data-north-event-page-") {
		t.Fatalf("parent event marker was not generated\n%s", markup)
	}
	known := map[string]componentView{"Dialog": {Name: "Dialog"}}
	if err := validateComponentReferences("Page", markup, known); err != nil {
		t.Fatalf("component event should not be treated as a Go prop: %v", err)
	}
	generated, err := compileComponent("routes", "Page", markup, known)
	if err != nil {
		t.Fatal(err)
	}
	result := string(generated)
	if !strings.Contains(result, `north-event-scope style=\"display:contents\" data-north-event-page-`) || !strings.Contains(result, `RenderDialog`) || !strings.Contains(result, `/north-event-scope`) {
		t.Fatalf("generated parent does not preserve the event boundary\n%s", result)
	}

	childSource := []byte(`<script lang="ts">
function close(): void { dispatch("close", { reason: "done" }); }
</script>
<button on:click={close}>Close</button>`)
	_, childModule, err := compileClientComponentWithOptions("Dialog", childSource, clientCompileOptions{Scoped: true})
	if err != nil {
		t.Fatal(err)
	}
	if childModule == nil || !strings.Contains(string(childModule.Source), "CustomEvent") || !strings.Contains(string(childModule.Source), "bubbles: true") {
		t.Fatalf("child dispatch helper was not compiled\n%s", childModule.Source)
	}
}

func TestComponentEventWithFollowingPropsCompiles(t *testing.T) {
	parentSource := []byte(`<script lang="ts">
function toggleSidebar(): void {}
</script>
<LayoutTopNavigation on:toggle-sidebar={toggleSidebar} DisplayName={Props.DisplayName} NotificationCount={Props.NotificationCount} />`)
	markup, _, err := compileClientComponentWithOptions("DashboardLayout", parentSource, clientCompileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	known := map[string]componentView{"LayoutTopNavigation": {
		Name:  "LayoutTopNavigation",
		Props: []prop{{Name: "DisplayName", Type: "string"}, {Name: "NotificationCount", Type: "int"}},
	}}
	if err := validateComponentReferences("DashboardLayout", markup, known); err != nil {
		t.Fatalf("component event marker before regular props should compile: %v\n%s", err, markup)
	}
}

func TestComponentEventMarkersCannotImpersonateArbitraryAttributes(t *testing.T) {
	for _, name := range []string{"data-north-event-page-0", "data-north-event-dialog-close"} {
		if !componentEventAttribute(name) {
			t.Errorf("%q should be accepted", name)
		}
	}
	for _, name := range []string{"data-north-event-", "data-north-event-x onclick", "data-other-event-x"} {
		if componentEventAttribute(name) {
			t.Errorf("%q should be rejected", name)
		}
	}
}
