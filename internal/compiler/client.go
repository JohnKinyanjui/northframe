package compiler

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/JohnKinyanjui/northframe/internal/dependencies"
	"github.com/evanw/esbuild/pkg/api"
)

var (
	typeScriptBlock = regexp.MustCompile(`(?s)<script\s+lang=["']ts["']\s*>(.*?)</script>`)
	clientEvent     = regexp.MustCompile(`on:([A-Za-z][A-Za-z0-9_-]*)=\{([^{}]+)\}`)
	clientShow      = regexp.MustCompile(`show=(?:\{#([A-Za-z_][A-Za-z0-9_.]*)\}|#\{([A-Za-z_][A-Za-z0-9_.]*)\})`)
	clientModel     = regexp.MustCompile(`bind:value=(?:\{#([A-Za-z_][A-Za-z0-9_.]*)\}|#\{([A-Za-z_][A-Za-z0-9_.]*)\})`)
	clientClass     = regexp.MustCompile(`class:([A-Za-z_][A-Za-z0-9_-]*)=(?:\{#([A-Za-z_][A-Za-z0-9_.]*)\}|#\{([A-Za-z_][A-Za-z0-9_.]*)\})`)
	clientAttr      = regexp.MustCompile(`(aria-[A-Za-z0-9_-]+|data-[A-Za-z0-9_-]+|title)=(?:\{#([A-Za-z_][A-Za-z0-9_.]*)\}|#\{([A-Za-z_][A-Za-z0-9_.]*)\})`)
	clientText      = regexp.MustCompile(`(?:\{#([A-Za-z_][A-Za-z0-9_.]*)\}|#\{([A-Za-z_][A-Za-z0-9_.]*)\})`)
	clientImport    = regexp.MustCompile(`(?ms)^[ \t]*import(?:[ \t]+type)?(?:[ \t]+(?:[^;"']|"[^"\n]*"|'[^'\n]*')*?[ \t]+from[ \t]+)?[ \t]*["'][^"'\n]+["'][ \t]*;?[ \t]*(?:\r?\n|$)`)
	clientExport    = regexp.MustCompile(`(?m)^\s*export\b`)
)

type clientModule struct {
	Path   string
	Source []byte
}

type clientBinding struct {
	Selector   string
	Kind       string
	Expression string
	Name       string
}

type clientEventBinding struct {
	Selector  string
	Event     string
	HandlerTS string
}

type clientCompileOptions struct {
	Contract   string
	PropsType  string
	Props      []prop
	Scoped     bool
	SourcePath string
	Project    dependencies.Project
}

func firstClientExpression(parts []string, indexes ...int) string {
	for _, index := range indexes {
		if index < len(parts) && parts[index] != "" {
			return parts[index]
		}
	}
	return ""
}

func compileClientComponent(componentName string, source []byte) ([]byte, *clientModule, error) {
	return compileClientComponentWithOptions(componentName, source, clientCompileOptions{})
}

func compileClientComponentWithOptions(componentName string, source []byte, options clientCompileOptions) ([]byte, *clientModule, error) {
	markup := string(source)
	interfaceContracts := append(frontmatterPropsBlock.FindAllString(markup, -1), interfacePropsBlock.FindAllString(markup, -1)...)
	if len(interfaceContracts) > 1 {
		return nil, nil, fmt.Errorf("%s contains more than one Props contract", componentName)
	}
	interfaceContract := ""
	if len(interfaceContracts) == 1 {
		interfaceContract = interfaceContracts[0]
		markup = frontmatterPropsBlock.ReplaceAllString(markup, interfacePropsSentinel)
		markup = interfacePropsBlock.ReplaceAllString(markup, interfacePropsSentinel)
	}
	restoreInterfaceContract := func(value string) string {
		value = strings.ReplaceAll(value, interfacePropsSentinel, "")
		return interfaceContract + value
	}
	matches := typeScriptBlock.FindAllStringSubmatch(markup, -1)
	if len(matches) > 1 {
		return nil, nil, fmt.Errorf("%s contains more than one <script lang=\"ts\"> block", componentName)
	}
	typeScript := ""
	if len(matches) == 1 {
		typeScript = strings.TrimSpace(matches[0][1])
		markup = typeScriptBlock.ReplaceAllString(markup, "")
	}

	prefix := clientPrefix(componentName)
	bindings := make([]clientBinding, 0)
	events := make([]clientEventBinding, 0)
	nextMarker := func(kind string) string {
		return fmt.Sprintf("data-north-%s-%s-%d", kind, prefix, len(bindings)+len(events))
	}

	markup = clientEvent.ReplaceAllStringFunc(markup, func(raw string) string {
		parts := clientEvent.FindStringSubmatch(raw)
		marker := nextMarker("event")
		events = append(events, clientEventBinding{Selector: "[" + marker + "]", Event: parts[1], HandlerTS: strings.TrimSpace(parts[2])})
		return marker
	})
	markup = clientShow.ReplaceAllStringFunc(markup, func(raw string) string {
		expression := firstClientExpression(clientShow.FindStringSubmatch(raw), 1, 2)
		marker := nextMarker("bind")
		bindings = append(bindings, clientBinding{Selector: "[" + marker + "]", Kind: "show", Expression: expression})
		return marker + " hidden"
	})
	markup = clientModel.ReplaceAllStringFunc(markup, func(raw string) string {
		expression := firstClientExpression(clientModel.FindStringSubmatch(raw), 1, 2)
		marker := nextMarker("bind")
		bindings = append(bindings, clientBinding{Selector: "[" + marker + "]", Kind: "model", Expression: expression})
		return marker
	})
	markup = clientClass.ReplaceAllStringFunc(markup, func(raw string) string {
		parts := clientClass.FindStringSubmatch(raw)
		marker := nextMarker("bind")
		bindings = append(bindings, clientBinding{Selector: "[" + marker + "]", Kind: "class", Name: parts[1], Expression: firstClientExpression(parts, 2, 3)})
		return marker
	})
	markup = clientAttr.ReplaceAllStringFunc(markup, func(raw string) string {
		parts := clientAttr.FindStringSubmatch(raw)
		marker := nextMarker("bind")
		bindings = append(bindings, clientBinding{Selector: "[" + marker + "]", Kind: "attribute", Name: parts[1], Expression: firstClientExpression(parts, 2, 3)})
		return parts[1] + `="false" ` + marker
	})
	markup = clientText.ReplaceAllStringFunc(markup, func(raw string) string {
		expression := firstClientExpression(clientText.FindStringSubmatch(raw), 1, 2)
		marker := nextMarker("bind")
		bindings = append(bindings, clientBinding{Selector: "[" + marker + "]", Kind: "text", Expression: expression})
		return "<span " + marker + "></span>"
	})

	if len(bindings) == 0 && len(events) == 0 && typeScript == "" {
		return []byte(restoreInterfaceContract(markup)), nil, nil
	}
	if typeScript == "" {
		return nil, nil, fmt.Errorf("%s uses browser state but has no <script lang=\"ts\"> block", componentName)
	}
	if regexp.MustCompile(`(?m)^\s*(let|const|var)\s+props\b`).MatchString(typeScript) {
		return nil, nil, fmt.Errorf("%s TypeScript cannot declare reserved value props", componentName)
	}
	if clientExport.MatchString(typeScript) {
		return nil, nil, fmt.Errorf("%s TypeScript cannot export from a .north script; move shared exports into $client", componentName)
	}
	if err := checkClientTypes(componentName, typeScript, bindings, events, options.Props); err != nil {
		return nil, nil, err
	}

	imports, clientBody := extractClientImports(typeScript)
	moduleSource := buildClientModule(imports, clientBody, bindings, events, prefix, options)
	compiled, err := bundleClientModule(componentName, moduleSource, options)
	if err != nil {
		return nil, nil, err
	}
	digest := sha256.Sum256(compiled)
	modulePath := fmt.Sprintf("%s-%x.js", prefix, digest[:6])
	moduleTag := `<script type="module" src="/_northframe/components/` + modulePath + `"></script>`
	if options.Scoped {
		markup = `<north-component data-north-scope="` + prefix + `">` + propsSentinel + markup + `</north-component>` + moduleTag
	} else if strings.Contains(markup, "</body>") {
		markup = strings.Replace(markup, "</body>", propsSentinel+moduleTag+"</body>", 1)
	} else {
		markup += propsSentinel + moduleTag
	}
	return []byte(restoreInterfaceContract(markup)), &clientModule{Path: modulePath, Source: compiled}, nil
}

func bundleClientModule(componentName, source string, options clientCompileOptions) ([]byte, error) {
	if options.Project.Root == "" && options.SourcePath == "" {
		result := api.Transform(source, api.TransformOptions{
			Loader: api.LoaderTS, Format: api.FormatESModule, Target: api.ES2020,
			Sourcefile: componentName + ".north.ts", LegalComments: api.LegalCommentsNone,
		})
		if len(result.Errors) > 0 {
			return nil, fmt.Errorf("compile %s TypeScript: %s", componentName, formatClientMessages(result.Errors))
		}
		return result.Code, nil
	}
	workingDirectory := options.Project.Root
	if workingDirectory == "" {
		workingDirectory, _ = os.Getwd()
	}
	resolveDirectory := workingDirectory
	if options.SourcePath != "" {
		resolveDirectory = filepath.Dir(options.SourcePath)
	}
	result := api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents: source, ResolveDir: resolveDirectory,
			Sourcefile: componentName + ".north.ts", Loader: api.LoaderTS,
		},
		AbsWorkingDir: workingDirectory,
		Bundle:        true, Write: false, Outfile: "client.js",
		Format: api.FormatESModule, Platform: api.PlatformBrowser, Target: api.ES2020,
		LegalComments: api.LegalCommentsNone,
		NodePaths:     []string{options.Project.NodeModules},
		External:      []string{"/_northframe/component.js", "http://*", "https://*"},
		Plugins:       []api.Plugin{clientResolverPlugin(options.Project)},
	})
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("compile %s TypeScript: %s", componentName, formatClientMessages(result.Errors))
	}
	var javascript []byte
	for _, output := range result.OutputFiles {
		switch strings.ToLower(filepath.Ext(output.Path)) {
		case ".js":
			javascript = output.Contents
		case ".css":
			return nil, fmt.Errorf("compile %s TypeScript: CSS imports from JavaScript packages are not supported yet; use page.css or layout.css", componentName)
		}
	}
	if len(javascript) == 0 {
		return nil, fmt.Errorf("compile %s TypeScript: bundler produced no JavaScript", componentName)
	}
	return javascript, nil
}

func clientResolverPlugin(project dependencies.Project) api.Plugin {
	return api.Plugin{Name: "northframe-imports", Setup: func(build api.PluginBuild) {
		build.OnResolve(api.OnResolveOptions{Filter: `.*`}, func(arguments api.OnResolveArgs) (api.OnResolveResult, error) {
			importPath := arguments.Path
			if importPath == "/_northframe/component.js" || strings.HasPrefix(importPath, "http://") || strings.HasPrefix(importPath, "https://") {
				return api.OnResolveResult{Path: importPath, External: true}, nil
			}
			if strings.HasPrefix(importPath, "$client/") || importPath == "$client" {
				path, err := resolveClientPath(project.ClientSource, strings.TrimPrefix(strings.TrimPrefix(importPath, "$client"), "/"))
				if err != nil {
					return api.OnResolveResult{}, err
				}
				return api.OnResolveResult{Path: path}, nil
			}
			if strings.HasPrefix(importPath, "node:") {
				return api.OnResolveResult{}, fmt.Errorf("Node built-in %q cannot run in the browser", importPath)
			}
			if bareClientImport(importPath) && !withinPath(arguments.Importer, project.NodeModules) {
				name := dependencies.PackageName(importPath)
				if _, declared := project.Dependencies[name]; !declared {
					return api.OnResolveResult{}, fmt.Errorf("package %q is not declared; run `north add %s`", name, name)
				}
			}
			return api.OnResolveResult{}, nil
		})
	}}
}

func bareClientImport(value string) bool {
	return value != "" && value[0] != '.' && value[0] != '/' && !strings.Contains(value, "://")
}

func resolveClientPath(root, relative string) (string, error) {
	if root == "" {
		return "", fmt.Errorf("$client imports require a client source directory")
	}
	base := filepath.Join(root, filepath.FromSlash(relative))
	if !withinPath(base, root) {
		return "", fmt.Errorf("$client import escapes %s", root)
	}
	candidates := []string{base, base + ".ts", base + ".tsx", base + ".js", base + ".jsx", filepath.Join(base, "index.ts"), filepath.Join(base, "index.tsx"), filepath.Join(base, "index.js"), filepath.Join(base, "index.jsx")}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("cannot resolve $client/%s", filepath.ToSlash(relative))
}

func withinPath(path, root string) bool {
	if path == "" || root == "" {
		return false
	}
	absolutePath, pathErr := filepath.Abs(path)
	absoluteRoot, rootErr := filepath.Abs(root)
	if pathErr != nil || rootErr != nil {
		return false
	}
	relative, err := filepath.Rel(absoluteRoot, absolutePath)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func extractClientImports(typeScript string) (string, string) {
	imports := clientImport.FindAllString(typeScript, -1)
	body := clientImport.ReplaceAllString(typeScript, "")
	return strings.TrimSpace(strings.Join(imports, "\n")), strings.TrimSpace(body)
}

func buildClientModule(imports, typeScript string, bindings []clientBinding, events []clientEventBinding, prefix string, options clientCompileOptions) string {
	var output strings.Builder
	output.WriteString(`import { mountComponent, readProps } from "/_northframe/component.js";` + "\n")
	if imports != "" {
		output.WriteString(imports)
		output.WriteString("\n")
	}
	output.WriteString("type NorthframeFieldErrors = Readonly<Record<string, string>>;\n")
	output.WriteString("type NorthframeActionResult<T = unknown> = Readonly<{ success: boolean; message?: string; errors?: NorthframeFieldErrors; data?: T; redirect?: string }>;\n")
	output.WriteString("type NorthframeActionEvent<T = unknown> = CustomEvent<{ response: Response; result: NorthframeActionResult<T> }>;\n")
	if options.Contract != "" {
		output.WriteString(options.Contract)
		output.WriteString("\n")
	}
	propsType := options.PropsType
	if propsType == "" {
		propsType = "Record<string, never>"
	}
	if options.Scoped {
		fmt.Fprintf(&output, "document.querySelectorAll(%s).forEach((root) => {\n", strconv.Quote(`[data-north-scope="`+prefix+`"]`))
	} else {
		output.WriteString("const root = document;\n")
	}
	output.WriteString("const dispatch = <T = unknown>(type: string, detail?: T): boolean => root.dispatchEvent(new CustomEvent<T>(type, { detail, bubbles: true }));\n")
	fmt.Fprintf(&output, "const props = readProps(root, %s) as %s;\n", strconv.Quote(prefix), propsType)
	output.WriteString(typeScript)
	output.WriteString("\nmountComponent({\n  bindings: [\n")
	for _, binding := range bindings {
		fmt.Fprintf(&output, "    { selector: %s, kind: %s, name: %s, read: () => (%s)", strconv.Quote(binding.Selector), strconv.Quote(binding.Kind), strconv.Quote(binding.Name), binding.Expression)
		if binding.Kind == "model" {
			fmt.Fprintf(&output, ", write: (value: any) => { %s = value; }", binding.Expression)
		}
		output.WriteString(" },\n")
	}
	output.WriteString("  ],\n  events: [\n")
	for _, event := range events {
		fmt.Fprintf(&output, "    { selector: %s, type: %s, run: (event: Event) => { const handler = (%s); return typeof handler === 'function' ? handler(event) : handler; } },\n", strconv.Quote(event.Selector), strconv.Quote(event.Event), event.HandlerTS)
	}
	output.WriteString("  ],\n}, root);\n")
	if options.Scoped {
		output.WriteString("});\n")
	}
	return output.String()
}

func clientPrefix(componentName string) string {
	return strings.ReplaceAll(toSnakeCase(componentName), "_", "-")
}

func formatClientMessages(messages []api.Message) string {
	formatted := api.FormatMessages(messages, api.FormatMessagesOptions{Kind: api.ErrorMessage, Color: false})
	return strings.TrimSpace(strings.Join(formatted, "\n"))
}
