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
		`north-component{display:contents}`,
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
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || filepath.Ext(path) != ".north" {
				return nil
			}
			source, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, match := range classAttribute.FindAllSubmatch(source, -1) {
				for _, className := range strings.Fields(string(match[1])) {
					if utilityRule(className) == "" {
						t.Errorf("%s uses unsupported utility %q", path, className)
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
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
	} {
		if declaration := arbitraryDeclaration(className); declaration != "" {
			t.Errorf("arbitraryDeclaration(%q) = %q, want empty", className, declaration)
		}
	}
}

func TestBuildStylesSupportsShellColorsAndOpacity(t *testing.T) {
	styles := string(BuildStyles([][]byte{[]byte(`<header class="bg-[#101918] text-white/80 border-white/10 bg-white/[0.12] ring-primary/70 from-gray-900 to-gray-100">`)}, nil))
	for _, expected := range []string{
		`.bg-\[\#101918\]{background-color:#101918}`,
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
