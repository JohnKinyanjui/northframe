package compiler

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/JohnKinyanjui/northframe/internal/dependencies"
)

// RouteBuild is the complete generated output for one convention-based routes tree.
type RouteBuild struct {
	Files map[string][]byte
	// RouteFiles is retained for source compatibility. Northframe no longer
	// writes generated files beside handwritten routes; all output lives below
	// the configured .generated directory.
	RouteFiles map[string][]byte
	RouteCount int
	Warnings   []string
}

type routeView struct {
	Directory        string
	Kind             string
	Name             string
	Path             string
	Source           []byte
	ImportPath       string
	ImportAlias      string
	HasActions       bool
	HasMiddleware    bool
	ClientModule     *clientModule
	Contract         string
	PropsType        string
	Props            []prop
	Imports          []componentImport
	SourceDir        string
	SourcePath       string
	PackageName      string
	GeneratedProps   bool
	PropsImportPath  string
	PropsImportAlias string
}

// BuildRoutes compiles web/routes/**/page.north and calls the typed loader in each
// colocated page.north.go/layout.north.go sidecar.
func BuildRoutes(routesDirectory, packageName, routeImportRoot string) (RouteBuild, error) {
	return buildProjectRoutes(routesDirectory, "", packageName, routeImportRoot, "", defaultGeneratedImportRoot(routeImportRoot))
}

// BuildProject compiles server-rendered views and an optional web/routes/api tree
// into one generated router package.
func BuildProject(routesDirectory, apiDirectory, packageName, routeImportRoot, apiImportRoot string) (RouteBuild, error) {
	return buildProjectRoutes(routesDirectory, apiDirectory, packageName, routeImportRoot, apiImportRoot, defaultGeneratedImportRoot(routeImportRoot))
}

// BuildProjectTo compiles a project into a caller-selected generated import
// root. The CLI uses this to keep all managed sources below .generated/routes.
func BuildProjectTo(routesDirectory, apiDirectory, packageName, routeImportRoot, apiImportRoot, generatedImportRoot string) (RouteBuild, error) {
	return buildProjectRoutes(routesDirectory, apiDirectory, packageName, routeImportRoot, apiImportRoot, generatedImportRoot)
}

func buildProjectRoutes(routesDirectory, apiDirectory, packageName, routeImportRoot, apiImportRoot, generatedImportRoot string) (RouteBuild, error) {
	clientProject, err := clientProjectFor(routesDirectory)
	if err != nil {
		return RouteBuild{}, err
	}
	views, customCSS, err := discoverRouteViews(routesDirectory, routeImportRoot)
	if err != nil {
		return RouteBuild{}, err
	}
	appCSS, err := readAppCSS(routesDirectory)
	if err != nil {
		return RouteBuild{}, err
	}
	if len(appCSS) > 0 {
		customCSS = append([][]byte{appCSS}, customCSS...)
	}
	if !identifier.MatchString(packageName) {
		return RouteBuild{}, fmt.Errorf("invalid package name %q", packageName)
	}
	errorPage, err := discoverErrorPage(routesDirectory)
	if err != nil {
		return RouteBuild{}, err
	}

	layouts, pages := groupRouteViews(views)
	if _, exists := layouts["."]; !exists {
		return RouteBuild{}, fmt.Errorf("web/routes/layout.north and web/routes/layout.north.go are required as the application shell")
	}
	if len(pages) == 0 {
		return RouteBuild{}, fmt.Errorf("no page.north files found below %s", routesDirectory)
	}
	routePropFiles, err := generateRoutePropFiles(views)
	if err != nil {
		return RouteBuild{}, err
	}
	apiRoutes, err := discoverAPIRoutes(apiDirectory, apiImportRoot)
	if err != nil {
		return RouteBuild{}, err
	}

	files := map[string][]byte{}
	for name, contents := range routePropFiles {
		files[name] = contents
	}
	styleSources := make([][]byte, 0, len(views))
	clientAssets := make([]publicAsset, 0)
	componentsRoot := filepath.Join(filepath.Dir(routesDirectory), "components")
	components, err := discoverComponents(componentsRoot)
	if err != nil {
		return RouteBuild{}, err
	}
	typeResolver, err := newTypeContractResolver(routesDirectory)
	if err != nil {
		return RouteBuild{}, err
	}
	for index := range components {
		components[index].Contract = typeResolver.contract(components[index].Name+"Props", components[index].Props, filepath.Dir(components[index].Path), components[index].Imports)
	}
	for index := range views {
		if views[index].GeneratedProps {
			views[index].PropsImportPath = generatedPropsImport(generatedImportRoot, views[index].Directory)
			views[index].PropsImportAlias = generatedPropsAlias(views[index].Directory)
		} else {
			views[index].PropsImportPath = views[index].ImportPath
			views[index].PropsImportAlias = views[index].ImportAlias
		}
		views[index].Contract = typeResolver.contract(views[index].PropsType, views[index].Props, views[index].SourceDir, views[index].Imports)
	}
	componentNames := make(map[string]componentView, len(components))
	for _, current := range components {
		componentNames[current.Name] = current
	}
	for index := range components {
		current := &components[index]
		current.Source, current.ClientModule, err = compileClientComponentWithOptions(current.Name, current.Source, clientCompileOptions{
			Contract: current.Contract, PropsType: current.Name + "Props", Props: current.Props, Scoped: true,
			SourcePath: current.Path, Project: clientProject,
		})
		if err != nil {
			return RouteBuild{}, fmt.Errorf("compile component %s: %w", current.Name, err)
		}
		if err := validateComponentReferences(current.Name, current.Source, componentNames); err != nil {
			return RouteBuild{}, err
		}
		generated, compileErr := compileComponent(packageName, current.Name, current.Source, componentNames)
		if compileErr != nil {
			return RouteBuild{}, fmt.Errorf("compile component %s: %w", current.Name, compileErr)
		}
		files["component_"+toSnakeCase(current.Name)+"_generated.go"] = generated
		styleSources = append(styleSources, current.Source)
		if current.ClientModule != nil {
			clientAssets = append(clientAssets, publicAsset{Path: current.ClientModule.Path, Content: current.ClientModule.Source, ContentType: "text/javascript; charset=utf-8"})
		}
	}
	if errorPage != nil {
		errorPage.Source, err = expandIncludes(errorPage.Source, componentsRoot, nil)
		if err != nil {
			return RouteBuild{}, fmt.Errorf("root error page: %w", err)
		}
		errorPage.Source, err = prepareErrorPage(errorPage.Source)
		if err != nil {
			return RouteBuild{}, err
		}
		errorPage.Contract = typeResolver.contract("ApplicationErrorProps", errorPage.Props, filepath.Dir(errorPage.Path), nil)
		errorPage.Source, errorPage.ClientModule, err = compileClientComponentWithOptions(errorPage.Name, errorPage.Source, clientCompileOptions{
			Contract: errorPage.Contract, PropsType: "ApplicationErrorProps", Props: errorPage.Props,
			SourcePath: errorPage.Path, Project: clientProject,
		})
		if err != nil {
			return RouteBuild{}, fmt.Errorf("compile root error page: %w", err)
		}
		if err := validateComponentReferences(errorPage.Name, errorPage.Source, componentNames); err != nil {
			return RouteBuild{}, err
		}
		generated, compileErr := compileComponent(packageName, errorPage.Name, errorPage.Source, componentNames)
		if compileErr != nil {
			return RouteBuild{}, fmt.Errorf("compile root error page: %w", compileErr)
		}
		files["application_error_generated.go"] = generated
		styleSources = append(styleSources, errorPage.Source)
		if errorPage.ClientModule != nil {
			clientAssets = append(clientAssets, publicAsset{Path: errorPage.ClientModule.Path, Content: errorPage.ClientModule.Source, ContentType: "text/javascript; charset=utf-8"})
		}
	}
	for index := range views {
		view := &views[index]
		view.Source, err = prepareRouteSource(*view, componentsRoot)
		if err != nil {
			return RouteBuild{}, err
		}
		if view.Kind == "layout" {
			layouts[view.Directory] = *view
		}
		view.Source, view.ClientModule, err = compileClientComponentWithOptions(view.Name, view.Source, clientCompileOptions{
			Contract: view.Contract, PropsType: view.PropsType, Props: view.Props,
			SourcePath: view.SourcePath, Project: clientProject, Scoped: view.Kind == "page",
		})
		if err != nil {
			return RouteBuild{}, fmt.Errorf("compile %s %s: %w", view.Directory, view.Kind, err)
		}
		if err := validateComponentReferences(view.Name, view.Source, componentNames); err != nil {
			return RouteBuild{}, err
		}
		if view.ClientModule != nil {
			clientAssets = append(clientAssets, publicAsset{Path: view.ClientModule.Path, Content: view.ClientModule.Source, ContentType: "text/javascript; charset=utf-8"})
		}
		generated, compileErr := compileRoute(packageName, view.Name, view.Source, view.PropsImportAlias, view.PropsImportPath, view.Kind, componentNames)
		if compileErr != nil {
			return RouteBuild{}, fmt.Errorf("compile %s %s: %w", view.Directory, view.Kind, compileErr)
		}
		files[toSnakeCase(view.Name)+"_generated.go"] = generated
		styleSources = append(styleSources, view.Source)
	}

	assets, err := discoverPublicAssets(filepath.Join(filepath.Dir(routesDirectory), "public"))
	if err != nil {
		return RouteBuild{}, err
	}
	classIssues := UnsupportedClasses(styleSources, customCSS)
	warnings := make([]string, 0, len(classIssues))
	for _, issue := range classIssues {
		warnings = append(warnings, fmt.Sprintf("unsupported utility class %q; define it in colocated CSS or use a supported Tailwind utility", issue.Name))
	}
	router, err := generateRouter(packageName, pages, layouts, apiRoutes, errorPage, BuildStyles(styleSources, customCSS), assets, clientAssets)
	if err != nil {
		return RouteBuild{}, err
	}
	files["router_generated.go"] = router
	files["northframe_contracts_generated.ts"] = buildContracts(components, views)
	return RouteBuild{Files: files, RouteFiles: map[string][]byte{}, RouteCount: len(pages) + apiMethodCount(apiRoutes), Warnings: warnings}, nil
}

func defaultGeneratedImportRoot(routeImportRoot string) string {
	trimmed := strings.TrimRight(routeImportRoot, "/")
	if strings.HasSuffix(trimmed, "/web/routes") {
		return strings.TrimSuffix(trimmed, "/web/routes") + "/.generated/routes"
	}
	if strings.HasSuffix(trimmed, "/routes") {
		return strings.TrimSuffix(trimmed, "/routes") + "/.generated/routes"
	}
	return trimmed + "/.generated/routes"
}

func generatedPropsImport(root, directory string) string {
	if directory == "." {
		return strings.TrimRight(root, "/") + "/root"
	}
	return strings.TrimRight(root, "/") + "/" + strings.Trim(directory, "/")
}

func generatedPropsAlias(directory string) string {
	if directory == "." {
		return "propsRoot"
	}
	return "props" + exportedName(directory)
}

func clientProjectFor(routesDirectory string) (dependencies.Project, error) {
	project, err := dependencies.InspectProject(routesDirectory)
	if err == nil {
		return project, nil
	}
	if !strings.Contains(err.Error(), "cannot find a Northframe project") {
		return dependencies.Project{}, err
	}
	absolute, absoluteErr := filepath.Abs(filepath.Dir(routesDirectory))
	if absoluteErr != nil {
		return dependencies.Project{}, absoluteErr
	}
	return dependencies.Project{
		Root: absolute, ClientSource: filepath.Join(absolute, "client"),
		NodeModules:  filepath.Join(absolute, ".northframe", "modules", "node_modules"),
		Dependencies: map[string]string{},
	}, nil
}

func buildContracts(components []componentView, views []routeView) []byte {
	var output strings.Builder
	output.WriteString("// Code generated by Northframe. DO NOT EDIT.\n\n")
	for _, current := range components {
		output.WriteString(current.Contract)
		output.WriteString("\n\n")
	}
	for _, view := range views {
		output.WriteString(view.Contract)
		output.WriteString("\n\n")
	}
	return []byte(output.String())
}

func groupRouteViews(views []routeView) (map[string]routeView, []routeView) {
	layouts := make(map[string]routeView)
	pages := make([]routeView, 0, len(views))
	for _, view := range views {
		if view.Kind == "layout" {
			layouts[view.Directory] = view
			continue
		}
		pages = append(pages, view)
	}
	return layouts, pages
}

func prepareRouteSource(view routeView, componentsRoot string) ([]byte, error) {
	source, err := expandIncludes(view.Source, componentsRoot, nil)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", view.Directory, view.Kind, err)
	}
	if view.Kind != "layout" {
		return source, nil
	}
	source, err = prepareLayout(source, view.Directory == ".")
	if err != nil {
		return nil, fmt.Errorf("%s layout: %w", view.Directory, err)
	}
	return source, nil
}
