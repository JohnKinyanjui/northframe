package docs

import (
	"bytes"
	"strings"
	"testing"

	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func TestRenderEscapesHTMLAndBuildsHeadingIndex(t *testing.T) {
	document := Render([]byte("# Hidden title\n\n## Start here\n\nUse **Go** and `north run`.\n\n- Fast\n\n```go\nif x < 2 {}\n```\n\n<script>alert(1)</script>"))
	var rendered bytes.Buffer
	if err := web.WriteHTML(&rendered, document.HTML); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`id="start-here"`, `<strong class=`, `<code class=`, `font-semibold text-rose-300`, `if</span> x &lt;`, `&lt;script&gt;`} {
		if !strings.Contains(rendered.String(), expected) {
			t.Fatalf("rendered HTML missing %q:\n%s", expected, rendered.String())
		}
	}
	if len(document.Headings) != 2 || document.Headings[1].ID != "start-here" {
		t.Fatalf("headings = %#v", document.Headings)
	}
}

func TestRenderHighlightsNorthframeCodeWithoutTrustingSourceHTML(t *testing.T) {
	document := Render([]byte("```north\n<h1>${Props.Title}</h1>\n<script>alert(1)</script>\n```"))
	var rendered bytes.Buffer
	_ = web.WriteHTML(&rendered, document.HTML)
	for _, expected := range []string{`data-language="north"`, `text-rose-300`, `text-orange-300`, `text-sky-300`, `&lt;`} {
		if !strings.Contains(rendered.String(), expected) {
			t.Fatalf("highlighted HTML missing %q:\n%s", expected, rendered.String())
		}
	}
	if strings.Contains(rendered.String(), `<script>alert`) {
		t.Fatalf("source HTML was not escaped: %s", rendered.String())
	}
}

func TestRenderDoesNotLinkUnsafeDestinations(t *testing.T) {
	document := Render([]byte("[Guide](/guide) [Bad](javascript:alert(1))"))
	var rendered bytes.Buffer
	_ = web.WriteHTML(&rendered, document.HTML)
	if strings.Contains(rendered.String(), `href="javascript:`) || !strings.Contains(rendered.String(), `href="/guide"`) {
		t.Fatalf("unexpected links: %s", rendered.String())
	}
}
