package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildStylesGeneratesUsedUtilitiesAndColocatedCSS(t *testing.T) {
	styles := string(BuildStyles(
		[][]byte{[]byte(`<main class="flex p-6 bg-slate-950 hover:bg-indigo-500 text-red-700">`)},
		[][]byte{[]byte(`.auth-form { display: contents; }`)},
	))

	for _, expected := range []string{
		`--nf-font-display:var(--nf-font-sans)`,
		`[hidden]{display:none!important}`,
		`north-component,north-route-segment{display:contents}`,
		`button,[type=button],[type=reset],[type=submit]{background-color:transparent;background-image:none}`,
		`blockquote,dl,dd,h1,h2,h3,h4,h5,h6,hr,figure,p,pre{margin:0}`,
		`.flex{display:flex}`,
		`.p-6{padding:1.5rem}`,
		`.bg-slate-950{background-color:#020617}`,
		`.hover\:bg-indigo-500:hover{background-color:#6366f1}`,
		`.text-red-700{color:#b91c1c}`,
		`.auth-form { display: contents; }`,
	} {
		if !strings.Contains(styles, expected) {
			t.Errorf("compiled CSS does not contain %q\n%s", expected, styles)
		}
	}
}

func TestExamplesOnlyUseSupportedUtilities(t *testing.T) {
	for _, root := range []string{"../../examples"} {
		var views [][]byte
		var customCSS [][]byte
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || (filepath.Ext(path) != ".north" && filepath.Ext(path) != ".css") {
				return nil
			}
			source, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if filepath.Ext(path) == ".north" {
				views = append(views, source)
			} else {
				customCSS = append(customCSS, source)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		for _, issue := range UnsupportedClasses(views, customCSS) {
			t.Errorf("examples use unsupported utility %q", issue.Name)
		}
	}
}

func TestUnsupportedClassesUnderstandsQuotesCustomCSSAndStructuralMarkers(t *testing.T) {
	source := []byte(`<div class="group peer custom-card flex made-up" data-label="it's fine"></div><style>.custom-card { color: red; }</style>`)
	issues := UnsupportedClasses([][]byte{source}, nil)
	if len(issues) != 1 || issues[0].Name != "made-up" {
		t.Fatalf("issues = %#v, want only made-up", issues)
	}
	if got := string(source[issues[0].Offset : issues[0].Offset+len(issues[0].Name)]); got != "made-up" {
		t.Fatalf("issue offset points to %q", got)
	}
}

func TestLiteralClassesAllowsSingleQuotesInsideDoubleQuotedClass(t *testing.T) {
	classes := literalClasses([]byte(`<span class="after:content-[''] flex"></span>`))
	if len(classes) != 2 || classes[0].Name != `after:content-['']` || classes[1].Name != "flex" {
		t.Fatalf("classes = %#v", classes)
	}
}

func TestBuildStylesUsesConfigurableFontVariables(t *testing.T) {
	styles := string(BuildStyles([][]byte{[]byte(`<h1 class="font-display">Title</h1>`)}, nil))
	if !strings.Contains(styles, `.font-display{font-family:var(--nf-font-display)}`) {
		t.Fatalf("font-display utility does not use the configurable variable\n%s", styles)
	}
}

func TestBuildStylesSupportsProductionAuthUtilities(t *testing.T) {
	styles := string(BuildStyles([][]byte{[]byte(`<main class="max-w-[380px] max-w-[400px] h-[25vh] xl:grid-cols-[minmax(0,1fr)_20rem] z-[70] bg-black/80 text-[11px] text-gray-500 focus:ring-2 focus:ring-[#dfff78] disabled:opacity-80 animate-[slot-scroll_20s_linear_infinite]">`)}, nil))
	for _, expected := range []string{
		`.max-w-\[380px\]{max-width:380px}`,
		`.max-w-\[400px\]{max-width:400px}`,
		`.h-\[25vh\]{height:25vh}`,
		`@media(min-width:1280px){.xl\:grid-cols-\[minmax\(0\,1fr\)_20rem\]{grid-template-columns:minmax(0,1fr) 20rem}}`,
		`.z-\[70\]{z-index:70}`,
		`.text-\[11px\]{font-size:11px}`,
		`.bg-black\/80{background-color:rgb(0 0 0/.8)}`,
		`.text-gray-500{color:#6b7280}`,
		`.focus\:ring-2:focus{box-shadow:0 0 0 2px var(--nf-ring-color)}`,
		`.disabled\:opacity-80:disabled{opacity:.8}`,
		`@keyframes slot-scroll`,
	} {
		if !strings.Contains(styles, expected) {
			t.Errorf("compiled CSS does not contain %q\n%s", expected, styles)
		}
	}
}

func TestArbitraryUtilitiesRejectRuleInjection(t *testing.T) {
	for _, className := range []string{
		`w-[1px;color:red]`,
		`h-[1px}]`,
		`max-w-[]`,
		`z-[front]`,
		`[animation-delay:-0.3s;color:red]`,
		`[background-image:url(javascript:alert(1))]`,
	} {
		if declaration := arbitraryDeclaration(className); declaration != "" {
			t.Errorf("arbitraryDeclaration(%q) = %q, want empty", className, declaration)
		}
	}
}

func TestArbitraryPropertyUtility(t *testing.T) {
	styles := string(BuildStyles([][]byte{[]byte(`<span class="[animation-delay:-0.3s]"></span>`)}, nil))
	if !strings.Contains(styles, `.\[animation-delay\:-0\.3s\]{animation-delay:-0.3s}`) {
		t.Fatalf("arbitrary property was not compiled\n%s", styles)
	}
}

func TestBuildStylesScansStaticClientClassAssignments(t *testing.T) {
	styles := string(BuildStyles([][]byte{[]byte(`<script lang="ts">const row = document.createElement("div"); row.className = "flex gap-3";</script>`)}, nil))
	for _, expected := range []string{`.flex{display:flex}`, `.gap-3{gap:.75rem}`} {
		if !strings.Contains(styles, expected) {
			t.Errorf("client-created class was not compiled: missing %q\n%s", expected, styles)
		}
	}
}

func TestBuildStylesSupportsShellColorsAndOpacity(t *testing.T) {
	styles := string(BuildStyles([][]byte{[]byte(`<header class="bg-[#101918] text-[#101918] text-white/80 border-white/10 bg-white/[0.12] ring-primary/70 from-gray-900 to-gray-100">`)}, nil))
	for _, expected := range []string{
		`.bg-\[\#101918\]{background-color:#101918}`,
		`.text-\[\#101918\]{color:#101918}`,
		`.text-white\/80{color:rgb(255 255 255 / 0.8)}`,
		`.border-white\/10{border-color:rgb(255 255 255 / 0.1)}`,
		`.bg-white\/\[0\.12\]{background-color:rgb(255 255 255 / 0.12)}`,
		`.ring-primary\/70{--nf-ring-color:rgb(212 245 66 / 0.7)}`,
		`.from-gray-900{--nf-gradient-from:#111827;`,
		`.to-gray-100{--nf-gradient-to:#f3f4f6}`,
	} {
		if !strings.Contains(styles, expected) {
			t.Errorf("compiled CSS does not contain %q\n%s", expected, styles)
		}
	}
}

func TestBuildStylesSupportsArbitraryAccentColor(t *testing.T) {
	source := []byte(`<input class="accent-[#253126]">`)
	styles := string(BuildStyles([][]byte{source}, nil))
	if !strings.Contains(styles, `.accent-\[\#253126\]{accent-color:#253126}`) {
		t.Fatalf("arbitrary accent color was not compiled\n%s", styles)
	}
	if issues := UnsupportedClasses([][]byte{source}, nil); len(issues) != 0 {
		t.Fatalf("unexpected unsupported utilities: %v", issues)
	}
}

func TestBuildStylesSupportsStandardLimePalette(t *testing.T) {
	styles := string(BuildStyles([][]byte{[]byte(`<div class="bg-lime-50 text-lime-700 border-lime-900"></div>`)}, nil))
	for _, expected := range []string{
		`.bg-lime-50{background-color:#f7fee7}`,
		`.text-lime-700{color:#4d7c0f}`,
		`.border-lime-900{border-color:#365314}`,
	} {
		if !strings.Contains(styles, expected) {
			t.Fatalf("missing lime palette utility %q\n%s", expected, styles)
		}
	}
}

func TestBuildStylesSupportsStandardOrangePalette(t *testing.T) {
	styles := string(BuildStyles([][]byte{[]byte(`<div class="bg-orange-50 text-orange-700 border-orange-200"></div>`)}, nil))
	for _, expected := range []string{
		`.bg-orange-50{background-color:#fff7ed}`,
		`.text-orange-700{color:#c2410c}`,
		`.border-orange-200{border-color:#fed7aa}`,
	} {
		if !strings.Contains(styles, expected) {
			t.Fatalf("missing orange palette utility %q\n%s", expected, styles)
		}
	}
}

func TestBuildStylesSupportsDashboardInteractionAndLayoutUtilities(t *testing.T) {
	styles := string(BuildStyles([][]byte{[]byte(`<div class="group space-y-5 divide-y divide-gray-100 first:pt-0 last:border-0"><label class="has-[:checked]:border-lime-300 has-[:checked]:bg-lime-50"><input type="checkbox"></label><button class="h-24 w-24 min-w-36 max-w-xs object-contain appearance-none resize-y opacity-0 group-hover:opacity-100 group-hover:translate-x-0.5 hover:shadow-md hover:brightness-95 backdrop-blur-[2px]"><span class="animate-bounce"></span></button></div>`)}, nil))
	for _, expected := range []string{
		`.space-y-5>:not([hidden])~:not([hidden]){margin-top:1.25rem}`,
		`.divide-y>:not([hidden])~:not([hidden]){border-top-width:1px}`,
		`.divide-gray-100>:not([hidden])~:not([hidden]){border-color:#f3f4f6}`,
		`.first\:pt-0:first-child{padding-top:0}`,
		`.last\:border-0:last-child{border-width:0}`,
		`.group:hover .group-hover\:opacity-100{opacity:1}`,
		`.group:hover .group-hover\:translate-x-0\.5{transform:translateX(.125rem)}`,
		`.has-\[\:checked\]\:border-lime-300:has(:checked){border-color:#bef264}`,
		`.has-\[\:checked\]\:bg-lime-50:has(:checked){background-color:#f7fee7}`,
		`.h-24{height:6rem}`,
		`.w-24{width:6rem}`,
		`.min-w-36{min-width:9rem}`,
		`.object-contain{object-fit:contain}`,
		`.hover\:shadow-md:hover{--nf-shadow-color:rgb(0 0 0/.1);box-shadow:0 4px 6px -1px var(--nf-shadow-color)}`,
		`.hover\:brightness-95:hover{filter:brightness(.95)}`,
		`.animate-bounce{animation:nf-bounce 1s infinite}`,
		`.backdrop-blur-\[2px\]{backdrop-filter:blur(2px);-webkit-backdrop-filter:blur(2px)}`,
	} {
		if !strings.Contains(styles, expected) {
			t.Errorf("compiled CSS does not contain %q\n%s", expected, styles)
		}
	}
}

func TestBuildStylesSupportsTailwindNumericSpacingScale(t *testing.T) {
	source := []byte(`<table class="min-w-180"></table>`)
	styles := string(BuildStyles([][]byte{source}, nil))
	if !strings.Contains(styles, `.min-w-180{min-width:45rem}`) {
		t.Fatalf("missing canonical numeric spacing utility\n%s", styles)
	}
	if issues := UnsupportedClasses([][]byte{source}, nil); len(issues) != 0 {
		t.Fatalf("unexpected unsupported utilities: %v", issues)
	}
}

func TestBuildStylesSupportsImageEditorTransitionsAndColors(t *testing.T) {
	styles := string(BuildStyles([][]byte{[]byte(`<button class="text-blue-400 transition-opacity"></button>`)}, nil))
	for _, expected := range []string{
		`.text-blue-400{color:#60a5fa}`,
		`.transition-opacity{transition-property:opacity;transition-duration:150ms}`,
	} {
		if !strings.Contains(styles, expected) {
			t.Errorf("compiled CSS does not contain %q\n%s", expected, styles)
		}
	}
}

func TestBuildStylesSupportsNamedOpenGroupsAndArbitrarySelectors(t *testing.T) {
	styles := string(BuildStyles([][]byte{[]byte(`<details class="group/nav"><summary class="[&::-webkit-details-marker]:hidden"><span class="group-open/nav:rotate-180"></span></summary></details>`)}, nil))
	for _, expected := range []string{
		`.group\/nav[open] .group-open\/nav\:rotate-180{transform:rotate(180deg)}`,
		`.\[\&\:\:-webkit-details-marker\]\:hidden::-webkit-details-marker{display:none}`,
	} {
		if !strings.Contains(styles, expected) {
			t.Errorf("compiled CSS does not contain %q\n%s", expected, styles)
		}
	}
}

func TestBuildStylesSupportsTailwindShadowColors(t *testing.T) {
	styles := string(BuildStyles([][]byte{[]byte(`<div class="shadow-inner shadow-white/5 shadow-xl shadow-black/20 shadow-sm shadow-lime-300/35"></div>`)}, nil))
	for _, expected := range []string{
		`.shadow-inner{--nf-shadow-color:rgb(0 0 0/.05);box-shadow:inset 0 2px 4px 0 var(--nf-shadow-color)}`,
		`.shadow-white\/5{--nf-shadow-color:rgb(255 255 255 / 0.05)}`,
		`.shadow-black\/20{--nf-shadow-color:rgb(0 0 0 / 0.2)}`,
		`.shadow-lime-300\/35{--nf-shadow-color:rgb(190 242 100 / 0.35)}`,
	} {
		if !strings.Contains(styles, expected) {
			t.Errorf("compiled CSS does not contain %q\n%s", expected, styles)
		}
	}
}

func TestBuildStylesSupportsAspectSquare(t *testing.T) {
	css := string(BuildStyles([][]byte{[]byte(`<div class="aspect-square"></div>`)}, nil))
	if !strings.Contains(css, ".aspect-square{aspect-ratio:1 / 1}") {
		t.Fatalf("expected aspect-square utility, got %q", css)
	}
}

func TestBuildStylesSupportsResponsiveFlexDirection(t *testing.T) {
	styles := string(BuildStyles([][]byte{[]byte(`<header class="flex flex-col sm:flex-row"></header>`)}, nil))
	for _, expected := range []string{
		`.flex-col{flex-direction:column}`,
		`@media(min-width:640px){.sm\:flex-row{flex-direction:row}}`,
	} {
		if !strings.Contains(styles, expected) {
			t.Errorf("compiled CSS does not contain %q\n%s", expected, styles)
		}
	}
}

func TestBuildStylesSupportsResponsiveTableCells(t *testing.T) {
	styles := string(BuildStyles([][]byte{[]byte(`<td class="hidden sm:table-cell md:table-cell lg:table-cell"></td>`)}, nil))
	for _, expected := range []string{
		`@media(min-width:640px){.sm\:table-cell{display:table-cell}}`,
		`@media(min-width:768px){.md\:table-cell{display:table-cell}}`,
		`@media(min-width:1024px){.lg\:table-cell{display:table-cell}}`,
	} {
		if !strings.Contains(styles, expected) {
			t.Errorf("compiled CSS does not contain %q\n%s", expected, styles)
		}
	}
}

func TestBuildStylesEmitsResponsiveSpacingAfterBaseSpacing(t *testing.T) {
	styles := string(BuildStyles([][]byte{[]byte(`<main class="p-5 lg:p-8"></main>`)}, nil))
	base := strings.Index(styles, `.p-5{padding:1.25rem}`)
	responsive := strings.Index(styles, `@media(min-width:1024px){.lg\:p-8{padding:2rem}}`)
	if base < 0 || responsive < 0 {
		t.Fatalf("missing spacing utilities\n%s", styles)
	}
	if responsive < base {
		t.Fatalf("responsive utility emitted before base utility\n%s", styles)
	}
}

func TestBuildStylesSupportsNegativeMarginSpacing(t *testing.T) {
	styles := string(BuildStyles([][]byte{[]byte(`<div class="-mt-2 -mx-4 hover:-mb-1"></div>`)}, nil))
	for _, expected := range []string{
		`.-mt-2{margin-top:-.5rem}`,
		`.-mx-4{margin-left:-1rem;margin-right:-1rem}`,
		`.hover\:-mb-1:hover{margin-bottom:-.25rem}`,
	} {
		if !strings.Contains(styles, expected) {
			t.Fatalf("missing negative spacing utility %q\n%s", expected, styles)
		}
	}
	if issues := UnsupportedClasses([][]byte{[]byte(`<div class="-mt-2 -mx-4 hover:-mb-1"></div>`)}, nil); len(issues) != 0 {
		t.Fatalf("unexpected unsupported negative spacing utilities: %v", issues)
	}
}

func TestBuildStylesEmitsDirectionalTwoPixelBorders(t *testing.T) {
	styles := string(BuildStyles([][]byte{[]byte(`<div class="border-t-2 border-r-2 border-b-2 border-l-2"></div>`)}, nil))
	for _, want := range []string{
		`.border-t-2{border-top-width:2px}`,
		`.border-r-2{border-right-width:2px}`,
		`.border-b-2{border-bottom-width:2px}`,
		`.border-l-2{border-left-width:2px}`,
	} {
		if !strings.Contains(styles, want) {
			t.Fatalf("expected generated styles to contain %q, got:\n%s", want, styles)
		}
	}
}

func TestBuildStylesSupportsArbitraryBorderWidths(t *testing.T) {
	source := []byte(`<div class="border-[0.5px] border-x-[0.125rem] border-t-[#101918]"></div>`)
	styles := string(BuildStyles([][]byte{source}, nil))
	for _, expected := range []string{
		`.border-\[0\.5px\]{border-width:0.5px}`,
		`.border-x-\[0\.125rem\]{border-left-width:0.125rem;border-right-width:0.125rem}`,
		`.border-t-\[\#101918\]{border-top-color:#101918}`,
	} {
		if !strings.Contains(styles, expected) {
			t.Fatalf("missing arbitrary border utility %q\n%s", expected, styles)
		}
	}
	if issues := UnsupportedClasses([][]byte{source}, nil); len(issues) != 0 {
		t.Fatalf("unexpected unsupported arbitrary border utilities: %v", issues)
	}
}

func TestBuildStylesSupportsStandardMaxWidths(t *testing.T) {
	styles := string(BuildStyles([][]byte{[]byte(`<div class="max-w-lg max-w-3xl max-w-5xl"></div>`)}, nil))
	for _, expected := range []string{
		`.max-w-lg{max-width:32rem}`,
		`.max-w-3xl{max-width:48rem}`,
		`.max-w-5xl{max-width:64rem}`,
	} {
		if !strings.Contains(styles, expected) {
			t.Fatalf("missing %q\n%s", expected, styles)
		}
	}
}

func TestBuildStylesSupportsElevenSpacingSize(t *testing.T) {
	styles := string(BuildStyles([][]byte{[]byte(`<span class="h-11 w-11"></span>`)}, nil))
	for _, expected := range []string{`.h-11{height:2.75rem}`, `.w-11{width:2.75rem}`} {
		if !strings.Contains(styles, expected) {
			t.Fatalf("missing %q\n%s", expected, styles)
		}
	}
}

func TestBuildStylesSupportsPointOfSaleUtilities(t *testing.T) {
	styles := string(BuildStyles([][]byte{[]byte(`<div class="aspect-[4/3] bg-[radial-gradient(circle_at_18%_0%,#ffffff_0,#eef1eb_100%)] shadow-[-14px_0_40px_rgba(30,45,25,0.04)] line-clamp-2 ring-1 ring-green-200 hover:-translate-y-0.5 2xl:grid-cols-5"></div>`)}, nil))
	for _, expected := range []string{
		`.aspect-\[4\/3\]{aspect-ratio:4/3}`,
		`background-image:radial-gradient(circle at 18% 0%,#ffffff 0,#eef1eb 100%)`,
		`box-shadow:-14px 0 40px rgba(30,45,25,0.04)`,
		`.line-clamp-2{display:-webkit-box;-webkit-box-orient:vertical;-webkit-line-clamp:2;overflow:hidden}`,
		`.ring-1{box-shadow:0 0 0 1px var(--nf-ring-color)}`,
		`.ring-green-200{--nf-ring-color:#bbf7d0}`,
		`.hover\:-translate-y-0\.5:hover{transform:translateY(-.125rem)}`,
		`@media(min-width:1536px){.\32 xl\:grid-cols-5{grid-template-columns:repeat(5,minmax(0,1fr))}}`,
	} {
		if !strings.Contains(styles, expected) {
			t.Errorf("compiled CSS does not contain %q\n%s", expected, styles)
		}
	}
}

func TestBuildStylesSupportsDocumentationSyntaxColors(t *testing.T) {
	source := []byte(`<code class="text-emerald-300 text-rose-300 text-sky-300 text-violet-300"></code>`)
	styles := string(BuildStyles([][]byte{source}, nil))
	for _, expected := range []string{
		`.text-emerald-300{color:#6ee7b7}`,
		`.text-rose-300{color:#fda4af}`,
		`.text-sky-300{color:#7dd3fc}`,
		`.text-violet-300{color:#c4b5fd}`,
	} {
		if !strings.Contains(styles, expected) {
			t.Errorf("compiled CSS does not contain %q\n%s", expected, styles)
		}
	}
}

func TestBuildStylesPreservesSpacesInArbitraryCalcValues(t *testing.T) {
	styles := string(BuildStyles([][]byte{[]byte(`<div class="h-[calc(100vh_-_4rem)]"></div>`)}, nil))
	if !strings.Contains(styles, `height:calc(100vh - 4rem)`) {
		t.Fatalf("arbitrary calc spacing was not preserved:\n%s", styles)
	}
}
