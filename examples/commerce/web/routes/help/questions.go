package help

import (
	"strings"

	"github.com/JohnKinyanjui/northframe/examples/commerce/internal/viewmodels"
)

var questionBank = []viewmodels.Question{
	{Category: "Positioning", Question: "Is Northframe Svelte rewritten in Go?", Answer: "No. Northframe borrows the colocated component experience, but Go performs routing, SSR, actions, middleware, database access, and deployment. TypeScript is reserved for browser interaction.", Owner: "Architecture"},
	{Category: "Positioning", Question: "How is Northframe different from SvelteKit?", Answer: "SvelteKit is a JavaScript application framework with Svelte's mature component compiler and ecosystem. Northframe is Go-first, emits native Go SSR renderers, integrates Go services directly, and deploys without a Node server.", Owner: "Architecture"},
	{Category: "Positioning", Question: "How is Northframe different from React or Next.js?", Answer: "Northframe does not ship a virtual DOM or require React hydration. It renders HTML in Go and generates small binding modules only for pages and components that declare browser state.", Owner: "Architecture"},
	{Category: "Positioning", Question: "When should I still choose SvelteKit?", Answer: "Choose SvelteKit when you need its mature component ecosystem, transitions, adapters, extensive tooling, or a JavaScript-first team. Northframe is strongest for Go-owned products and internal systems.", Owner: "Decision"},
	{Category: "Positioning", Question: "When is Northframe the better fit?", Answer: "Choose Northframe when Go already owns business logic and data, SSR matters, sqlc is valuable, and one compiled deployment artifact is preferable to separate frontend and backend projects.", Owner: "Decision"},
	{Category: "Runtime", Question: "Does Northframe require Node.js?", Answer: "No. Application authors use the northframe command and Go toolchain. Northframe performs TypeScript checking, JavaScript generation, CSS generation, routing, and asset embedding internally.", Owner: "Tooling"},
	{Category: "Runtime", Question: "Does a Northframe project need package.json?", Answer: "No. The northframe add command records browser dependencies in northframe.toml, locks exact tarballs and integrity hashes, and bundles imports without Node or an application package.json.", Owner: "Tooling"},
	{Category: "Runtime", Question: "Can a page import JavaScript date, chart, or UI libraries?", Answer: "Yes. Add a browser-compatible package with northframe add, then use a normal typed import in the page script. Northframe downloads, verifies, tree-shakes, and embeds it in the Go executable.", Owner: "Tooling"},
	{Category: "Runtime", Question: "Does Northframe use WebAssembly for state?", Answer: "No. Go owns server state and TypeScript compiles to JavaScript for immediate browser state. This avoids shipping the Go WebAssembly runtime for ordinary UI behaviour.", Owner: "Architecture"},
	{Category: "Runtime", Question: "Can Northframe support WebAssembly later?", Answer: "Yes, as an optional asset for CPU-heavy or offline features. It should not be the default state runtime because most interfaces need only small generated JavaScript bindings.", Owner: "Roadmap"},
	{Category: "State", Question: "Which state belongs in Go?", Answer: "Database records, sessions, authorization, validation, carts, durable workflow state, and anything shared between users belong in Go and its backing services.", Owner: "Architecture"},
	{Category: "State", Question: "Which state belongs in TypeScript?", Answer: "Dialogs, dropdowns, draft input, counters, optimistic feedback, loading indicators, and other short-lived browser interaction belong in the colocated TypeScript block.", Owner: "Architecture"},
	{Category: "State", Question: "How does TypeScript receive Go page data?", Answer: "Northframe generates a TypeScript contract from PageProps, safely serializes the SSR value into inert JSON, and exposes it as the typed props value inside the page module.", Owner: "Compiler"},
	{Category: "State", Question: "Is the TypeScript actually checked?", Answer: "Yes. Northframe checks its supported state subset for declaration, literal assignment, binding, event-handler, and PageProps mistakes before esbuild emits the browser module.", Owner: "Compiler"},
	{Category: "State", Question: "What do Props and hash expressions mean?", Answer: "A Go expression such as {Props.Title} is rendered during SSR. A hash expression such as {#open} reads reactive TypeScript state in the browser.", Owner: "Syntax"},
	{Category: "State", Question: "Can server state update without WebAssembly?", Answer: "Yes. A browser action can call Go over HTTP, receive HTML or a typed result, and update the page. SSE or WebSockets can push subsequent server changes.", Owner: "Architecture"},
	{Category: "Components", Question: "Are Northframe components independent?", Answer: "Yes. Files under components compile into their own Go props type and render function. Pages invoke them with PascalCase tags instead of copying their source into the route.", Owner: "Compiler"},
	{Category: "Components", Question: "Do components support typed props?", Answer: "Yes. A component declares Go props in its props block. Generated Go calls are compile-time checked, and a matching TypeScript contract is available to its browser script.", Owner: "Compiler"},
	{Category: "Components", Question: "Do components support slots?", Answer: "Yes. Child markup becomes a native web.Fragment passed through the generated component props. No server-side JavaScript renderer is involved.", Owner: "Compiler"},
	{Category: "Components", Question: "Does each component get isolated browser state?", Answer: "Yes. Northframe mounts every component instance against its own DOM scope, so opening one disclosure does not mutate another instance using the same component file.", Owner: "Runtime"},
	{Category: "Components", Question: "Can components import Go view models?", Answer: "Yes. Props blocks can import Go packages and use their types. The generated renderer references those types directly, so incompatible values fail during Go compilation.", Owner: "Compiler"},
	{Category: "SSR", Question: "How does server-side rendering work?", Answer: "The compiler converts each template into an io.Writer-based Go function. Routes call Page or Layout, receive typed props, and stream escaped HTML without executing JavaScript on the server.", Owner: "Architecture"},
	{Category: "SSR", Question: "Does the page work without JavaScript?", Answer: "SSR content, links, and native forms do. Features whose purpose is purely browser interaction degrade naturally or remain hidden until the generated module loads.", Owner: "Runtime"},
	{Category: "Routing", Question: "How are routes created?", Answer: "Folders below routes define URL structure. page.north renders the page, page.north.go loads data and declares actions, while layout.north wraps descendant routes.", Owner: "Convention"},
	{Category: "Routing", Question: "Does Northframe support JSON API routes?", Answer: "Yes. Exported HTTP method functions inside web/routes/api compile into /api routes. They receive the same web.Context, dependencies, middleware, parameters, and error handling as page actions without requiring a template.", Owner: "Convention"},
	{Category: "Routing", Question: "Can API routes use dynamic parameters?", Answer: "Yes. A web/routes/api/products/id_ folder compiles to /api/products/{id}, and GET can read the value with ctx.Param(\"id\"). Catch-all folders use the same name__ convention as page routes.", Owner: "Convention"},
	{Category: "Forms", Question: "How do POST actions work?", Answer: "A page sidecar registers method-and-path actions. Native forms post directly to Go, CSRF middleware validates unsafe requests, and enhanced forms add pending UI without replacing the server contract.", Owner: "Runtime"},
	{Category: "Database", Question: "Does Northframe require an ORM?", Answer: "No. Applications can use database/sql directly, but Northframe treats sqlc as the preferred typed workflow for PostgreSQL, MySQL, and SQLite.", Owner: "Data"},
	{Category: "Database", Question: "Can one application use PostgreSQL, MySQL, or SQLite?", Answer: "Yes. Northframe provides drivers and migration support for all three. sqlc configuration remains application-specific because SQL dialects and generated types differ.", Owner: "Data"},
	{Category: "Security", Question: "Are rendered values escaped?", Answer: "Yes. Normal server expressions pass through the Go HTML-escaping runtime. Serialized client props use safe JSON escaping so user content cannot terminate the inert script element.", Owner: "Security"},
	{Category: "Deployment", Question: "What gets deployed?", Answer: "One Go executable can contain generated route and component renderers, browser modules, utility CSS, public assets, migrations, and the application server.", Owner: "Operations"},
	{Category: "Deployment", Question: "How do I upgrade an application safely?", Answer: "Install the newer northframe command, preview managed changes with northframe upgrade --check, then run northframe upgrade. It rewrites only generated routes, validates a temporary Go binary, and restores the previous generated output if validation fails.", Owner: "Operations"},
	{Category: "Deployment", Question: "Is Northframe a reverse proxy between two servers?", Answer: "No. Go serves SSR pages, actions, framework assets, and public files from the same application server. There is no separate Svelte development or production server.", Owner: "Architecture"},
	{Category: "Honesty", Question: "Is Northframe production-complete today?", Answer: "No. It is an ambitious working prototype. Authentication conventions, full TypeScript semantics, component composition features, hot browser reload, observability, and deployment adapters still need hardening.", Owner: "Roadmap"},
	{Category: "Honesty", Question: "What is Northframe's main technical risk?", Answer: "It is building a compiler and framework surface at once. The project must keep syntax small, diagnostics excellent, generated code inspectable, and escape hatches available instead of imitating every JavaScript framework feature.", Owner: "Roadmap"},
}

func filterQuestions(query string) []viewmodels.Question {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return append([]viewmodels.Question(nil), questionBank...)
	}
	result := make([]viewmodels.Question, 0)
	for _, question := range questionBank {
		haystack := strings.ToLower(question.Category + " " + question.Question + " " + question.Answer + " " + question.Owner)
		if strings.Contains(haystack, query) {
			result = append(result, question)
		}
	}
	return result
}
