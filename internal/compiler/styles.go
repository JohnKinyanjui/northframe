package compiler

import (
	"bytes"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var classAttribute = regexp.MustCompile(`class=["']([^"']+)["']`)

const preflightCSS = `/* Northframe utility preflight */
:root{--nf-font-sans:ui-sans-serif,system-ui,sans-serif;--nf-font-serif:ui-serif,Georgia,serif;--nf-font-mono:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace;--nf-font-display:var(--nf-font-sans);--nf-ring-color:#dfff78}
*,::before,::after{box-sizing:border-box;border-width:0;border-style:solid}
html{line-height:1.5;-webkit-text-size-adjust:100%;font-family:var(--nf-font-sans)}
body{margin:0;line-height:inherit}button,input,select,textarea{font:inherit;color:inherit}
a{color:inherit;text-decoration:inherit}img,svg,video{display:block;max-width:100%;height:auto}north-component{display:contents}[hidden]{display:none!important}
@keyframes nf-spin{to{transform:rotate(360deg)}}
@keyframes slot-scroll{0%{transform:translateY(0)}100%{transform:translateY(-50%)}}
`

// BuildStyles compiles the Tailwind-compatible utility classes found in views
// and appends colocated page.css/layout.css source verbatim.
func BuildStyles(views [][]byte, customCSS [][]byte) []byte {
	classes := map[string]struct{}{}
	for _, view := range views {
		for _, match := range classAttribute.FindAllSubmatch(view, -1) {
			for _, className := range strings.Fields(string(match[1])) {
				classes[className] = struct{}{}
			}
		}
	}

	ordered := make([]string, 0, len(classes))
	for className := range classes {
		ordered = append(ordered, className)
	}
	sort.Strings(ordered)

	var output bytes.Buffer
	output.WriteString(preflightCSS)
	output.WriteString("/* Generated Northframe utilities */\n")
	for _, className := range ordered {
		if rule := utilityRule(className); rule != "" {
			output.WriteString(rule)
			output.WriteByte('\n')
		}
	}
	for _, css := range customCSS {
		if len(bytes.TrimSpace(css)) == 0 {
			continue
		}
		output.WriteString("\n/* Colocated route CSS */\n")
		output.Write(bytes.TrimSpace(css))
		output.WriteByte('\n')
	}
	return output.Bytes()
}

func utilityRule(className string) string {
	variant, base, found := strings.Cut(className, ":")
	if !found {
		base = className
		variant = ""
	}
	declaration := utilityDeclaration(base)
	if declaration == "" {
		return ""
	}
	selector := "." + escapeClass(className)
	switch variant {
	case "":
		return selector + "{" + declaration + "}"
	case "hover":
		return selector + ":hover{" + declaration + "}"
	case "focus":
		return selector + ":focus{" + declaration + "}"
	case "disabled":
		return selector + ":disabled{" + declaration + "}"
	case "placeholder":
		return selector + "::placeholder{" + declaration + "}"
	case "active":
		return selector + ":active{" + declaration + "}"
	case "sm":
		return "@media(min-width:640px){" + selector + "{" + declaration + "}}"
	case "md":
		return "@media(min-width:768px){" + selector + "{" + declaration + "}}"
	case "lg":
		return "@media(min-width:1024px){" + selector + "{" + declaration + "}}"
	case "xl":
		return "@media(min-width:1280px){" + selector + "{" + declaration + "}}"
	case "2xl":
		return "@media(min-width:1536px){" + selector + "{" + declaration + "}}"
	default:
		return ""
	}
}

func utilityDeclaration(className string) string {
	static := map[string]string{
		"block": "display:block", "inline-block": "display:inline-block", "inline": "display:inline", "hidden": "display:none",
		"flex": "display:flex", "inline-flex": "display:inline-flex", "grid": "display:grid", "contents": "display:contents",
		"relative": "position:relative", "absolute": "position:absolute", "fixed": "position:fixed", "sticky": "position:sticky", "isolate": "isolation:isolate",
		"inset-0": "inset:0", "top-0": "top:0", "right-0": "right:0", "bottom-0": "bottom:0", "left-0": "left:0", "top-1/2": "top:50%", "left-3": "left:.75rem", "right-3": "right:.75rem", "z-10": "z-index:10", "z-20": "z-index:20", "z-50": "z-index:50",
		"grid-cols-1": "grid-template-columns:repeat(1,minmax(0,1fr))", "grid-cols-2": "grid-template-columns:repeat(2,minmax(0,1fr))", "grid-cols-3": "grid-template-columns:repeat(3,minmax(0,1fr))", "grid-cols-4": "grid-template-columns:repeat(4,minmax(0,1fr))", "grid-cols-12": "grid-template-columns:repeat(12,minmax(0,1fr))",
		"col-span-1": "grid-column:span 1/span 1", "col-span-2": "grid-column:span 2/span 2", "col-span-3": "grid-column:span 3/span 3", "col-span-4": "grid-column:span 4/span 4", "col-span-5": "grid-column:span 5/span 5", "col-span-6": "grid-column:span 6/span 6", "col-span-8": "grid-column:span 8/span 8", "col-span-9": "grid-column:span 9/span 9", "col-span-10": "grid-column:span 10/span 10", "col-span-12": "grid-column:span 12/span 12",
		"flex-1": "flex:1 1 0%", "flex-col": "flex-direction:column", "flex-wrap": "flex-wrap:wrap", "grow": "flex-grow:1", "shrink-0": "flex-shrink:0",
		"items-start": "align-items:flex-start", "items-center": "align-items:center", "items-end": "align-items:flex-end", "place-items-center": "place-items:center",
		"justify-start": "justify-content:flex-start", "justify-center": "justify-content:center", "justify-end": "justify-content:flex-end", "justify-between": "justify-content:space-between",
		"text-left": "text-align:left", "text-center": "text-align:center", "text-right": "text-align:right", "text-start": "text-align:start",
		"font-sans": "font-family:var(--nf-font-sans)", "font-serif": "font-family:var(--nf-font-serif)", "font-mono": "font-family:var(--nf-font-mono)", "font-display": "font-family:var(--nf-font-display)", "font-jakarta": "font-family:'Plus Jakarta Sans','Salesforce Sans',ui-sans-serif,system-ui,sans-serif",
		"font-normal": "font-weight:400", "font-medium": "font-weight:500", "font-semibold": "font-weight:600", "font-bold": "font-weight:700", "font-extrabold": "font-weight:800",
		"uppercase": "text-transform:uppercase", "lowercase": "text-transform:lowercase", "tracking-tight": "letter-spacing:-.025em", "tracking-wide": "letter-spacing:.025em", "tracking-widest": "letter-spacing:.1em",
		"leading-none": "line-height:1", "leading-tight": "line-height:1.25", "leading-relaxed": "line-height:1.625", "tabular-nums": "font-variant-numeric:tabular-nums", "antialiased": "-webkit-font-smoothing:antialiased;-moz-osx-font-smoothing:grayscale",
		"rounded": "border-radius:.25rem", "rounded-md": "border-radius:.375rem", "rounded-lg": "border-radius:.5rem", "rounded-xl": "border-radius:.75rem", "rounded-2xl": "border-radius:1rem", "rounded-3xl": "border-radius:1.5rem", "rounded-full": "border-radius:9999px", "rounded-t-xl": "border-top-left-radius:.75rem;border-top-right-radius:.75rem",
		"border": "border-width:1px", "border-2": "border-width:2px", "border-0": "border-width:0", "border-t": "border-top-width:1px", "border-r": "border-right-width:1px", "border-b": "border-bottom-width:1px", "border-l": "border-left-width:1px", "border-dashed": "border-style:dashed",
		"shadow-sm": "box-shadow:0 1px 2px rgb(15 23 42/.06)", "shadow": "box-shadow:0 1px 3px rgb(0 0 0/.1)", "shadow-inner": "box-shadow:inset 0 2px 4px 0 rgb(0 0 0/.05)", "shadow-lg": "box-shadow:0 10px 15px -3px rgb(0 0 0/.1)", "shadow-xl": "box-shadow:0 20px 25px -5px rgb(0 0 0/.1)", "shadow-2xl": "box-shadow:0 25px 50px -12px rgb(0 0 0/.5)",
		"w-full": "width:100%", "w-screen": "width:100vw", "w-auto": "width:auto", "w-2": "width:.5rem", "w-3": "width:.75rem", "w-3.5": "width:.875rem", "w-8": "width:2rem", "w-9": "width:2.25rem", "w-10": "width:2.5rem", "w-12": "width:3rem", "w-64": "width:16rem",
		"h-full": "height:100%", "h-screen": "height:100vh", "h-0.5": "height:.125rem", "h-2": "height:.5rem", "h-3": "height:.75rem", "h-3.5": "height:.875rem", "h-8": "height:2rem", "h-9": "height:2.25rem", "h-10": "height:2.5rem", "h-12": "height:3rem", "h-16": "height:4rem", "h-20": "height:5rem", "h-[20vh]": "height:20vh", "h-[25vh]": "height:25vh", "h-[40vh]": "height:40vh",
		"min-h-0": "min-height:0", "min-h-screen": "min-height:100vh", "min-w-0": "min-width:0", "max-w-sm": "max-width:24rem", "max-w-[380px]": "max-width:380px", "max-w-md": "max-width:28rem", "max-w-xl": "max-width:36rem", "max-w-2xl": "max-width:42rem", "max-w-4xl": "max-width:56rem", "max-w-6xl": "max-width:72rem", "max-w-7xl": "max-width:80rem",
		"mx-auto": "margin-left:auto;margin-right:auto", "overflow-hidden": "overflow:hidden", "overflow-x-auto": "overflow-x:auto", "overflow-y-auto": "overflow-y:auto", "cursor-pointer": "cursor:pointer", "cursor-not-allowed": "cursor:not-allowed", "pointer-events-none": "pointer-events:none", "select-none": "user-select:none", "whitespace-nowrap": "white-space:nowrap", "truncate": "overflow:hidden;text-overflow:ellipsis;white-space:nowrap", "object-cover": "object-fit:cover", "list-none": "list-style-type:none",
		"outline-none": "outline:2px solid transparent;outline-offset:2px", "scale-95": "transform:scale(.95)", "-translate-y-1/2": "transform:translateY(-50%)", "opacity-0": "opacity:0", "opacity-30": "opacity:.3", "opacity-40": "opacity:.4", "opacity-50": "opacity:.5", "opacity-60": "opacity:.6", "opacity-70": "opacity:.7", "opacity-80": "opacity:.8", "opacity-100": "opacity:1",
		"transition": "transition-property:color,background-color,border-color,opacity,transform;transition-duration:150ms", "transition-all": "transition-property:all;transition-duration:150ms", "transition-colors": "transition-property:color,background-color,border-color;transition-duration:150ms", "duration-100": "transition-duration:100ms", "duration-200": "transition-duration:200ms", "duration-300": "transition-duration:300ms",
		"animate-spin": "animation:nf-spin 1s linear infinite", "animate-[slot-scroll_20s_linear_infinite]": "animation:slot-scroll 20s linear infinite", "animate-[slot-scroll_22s_linear_infinite]": "animation:slot-scroll 22s linear infinite", "animate-[slot-scroll_25s_linear_infinite]": "animation:slot-scroll 25s linear infinite", "animate-[slot-scroll_28s_linear_infinite]": "animation:slot-scroll 28s linear infinite",
		"ring-2": "box-shadow:0 0 0 2px var(--nf-ring-color)", "ring-[#dfff78]": "--nf-ring-color:#dfff78", "border-[#8cc900]": "border-color:#8cc900", "border-t-white": "border-top-color:#fff", "border-white/40": "border-color:rgb(255 255 255/.4)",
		"bg-black/80": "background-color:rgb(0 0 0/.8)", "bg-black/90": "background-color:rgb(0 0 0/.9)", "bg-primary": "background-color:#d4f542", "bg-primary/90": "background-color:rgb(212 245 66/.9)", "text-primary-foreground": "color:#101918",
		"bg-linear-to-b": "background-image:linear-gradient(to bottom,var(--nf-gradient-stops))", "bg-linear-to-t": "background-image:linear-gradient(to top,var(--nf-gradient-stops))", "bg-linear-to-br": "background-image:linear-gradient(to bottom right,var(--nf-gradient-stops))", "from-black": "--nf-gradient-from:#000;--nf-gradient-to:rgb(0 0 0/0);--nf-gradient-stops:var(--nf-gradient-from),var(--nf-gradient-to)", "from-black/80": "--nf-gradient-from:rgb(0 0 0/.8);--nf-gradient-to:rgb(0 0 0/0);--nf-gradient-stops:var(--nf-gradient-from),var(--nf-gradient-to)", "via-black/90": "--nf-gradient-stops:var(--nf-gradient-from),rgb(0 0 0/.9),var(--nf-gradient-to)", "to-transparent": "--nf-gradient-to:transparent",
		"backdrop-blur-sm": "backdrop-filter:blur(4px);-webkit-backdrop-filter:blur(4px)",
		"sr-only":          "position:absolute;width:1px;height:1px;padding:0;margin:-1px;overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap;border-width:0",
	}
	if declaration := static[className]; declaration != "" {
		return declaration
	}
	if declaration := arbitraryDeclaration(className); declaration != "" {
		return declaration
	}
	if declaration := spacingDeclaration(className); declaration != "" {
		return declaration
	}
	if strings.HasPrefix(className, "gap-") {
		if value := spacingValue(strings.TrimPrefix(className, "gap-")); value != "" {
			return "gap:" + value
		}
	}
	if strings.HasPrefix(className, "text-") {
		value := strings.TrimPrefix(className, "text-")
		if size := textSize(value); size != "" {
			return size
		}
		if color := colorValue(value); color != "" {
			return "color:" + color
		}
	}
	if strings.HasPrefix(className, "bg-") {
		if color := colorValue(strings.TrimPrefix(className, "bg-")); color != "" {
			return "background-color:" + color
		}
	}
	if strings.HasPrefix(className, "border-") {
		if color := colorValue(strings.TrimPrefix(className, "border-")); color != "" {
			return "border-color:" + color
		}
	}
	if strings.HasPrefix(className, "ring-") {
		if color := colorValue(strings.TrimPrefix(className, "ring-")); color != "" {
			return "--nf-ring-color:" + color
		}
	}
	if strings.HasPrefix(className, "from-") {
		if color := colorValue(strings.TrimPrefix(className, "from-")); color != "" {
			return "--nf-gradient-from:" + color + ";--nf-gradient-to:transparent;--nf-gradient-stops:var(--nf-gradient-from),var(--nf-gradient-to)"
		}
	}
	if strings.HasPrefix(className, "via-") {
		if color := colorValue(strings.TrimPrefix(className, "via-")); color != "" {
			return "--nf-gradient-stops:var(--nf-gradient-from)," + color + ",var(--nf-gradient-to)"
		}
	}
	if strings.HasPrefix(className, "to-") {
		if color := colorValue(strings.TrimPrefix(className, "to-")); color != "" {
			return "--nf-gradient-to:" + color
		}
	}
	return ""
}

// arbitraryDeclaration supports Tailwind's bracket syntax for layout values
// without requiring projects to install Node.js or the Tailwind CLI. Values are
// limited to a safe CSS token subset so source classes cannot terminate a rule.
func arbitraryDeclaration(className string) string {
	prefixes := []struct {
		prefix   string
		property string
	}{
		{"max-w-", "max-width"},
		{"max-h-", "max-height"},
		{"min-w-", "min-width"},
		{"min-h-", "min-height"},
		{"w-", "width"},
		{"h-", "height"},
		{"top-", "top"},
		{"right-", "right"},
		{"bottom-", "bottom"},
		{"left-", "left"},
		{"inset-", "inset"},
		{"rounded-", "border-radius"},
		{"tracking-", "letter-spacing"},
		{"leading-", "line-height"},
		{"grid-cols-", "grid-template-columns"},
		{"grid-rows-", "grid-template-rows"},
		{"basis-", "flex-basis"},
	}

	for _, candidate := range prefixes {
		if !strings.HasPrefix(className, candidate.prefix) {
			continue
		}
		value, ok := bracketValue(strings.TrimPrefix(className, candidate.prefix))
		if ok {
			return candidate.property + ":" + value
		}
	}

	if value, ok := bracketValue(strings.TrimPrefix(className, "text-")); ok && strings.HasPrefix(className, "text-") {
		return "font-size:" + value
	}
	if value, ok := bracketValue(strings.TrimPrefix(className, "z-")); ok && strings.HasPrefix(className, "z-") {
		if _, err := strconv.Atoi(value); err == nil {
			return "z-index:" + value
		}
	}
	return ""
}

func bracketValue(value string) (string, bool) {
	if len(value) < 3 || value[0] != '[' || value[len(value)-1] != ']' {
		return "", false
	}
	value = strings.ReplaceAll(value[1:len(value)-1], "_", " ")
	if strings.TrimSpace(value) == "" || strings.ContainsAny(value, ";{}\\\n\r") {
		return "", false
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			strings.ContainsRune(" .,%#()+*/:-", character) {
			continue
		}
		return "", false
	}
	return value, true
}

func spacingDeclaration(className string) string {
	prefixes := []struct {
		prefix, properties string
	}{
		{"px-", "padding-left:%s;padding-right:%s"}, {"py-", "padding-top:%s;padding-bottom:%s"},
		{"pt-", "padding-top:%s"}, {"pr-", "padding-right:%s"}, {"pb-", "padding-bottom:%s"}, {"pl-", "padding-left:%s"}, {"p-", "padding:%s"},
		{"mx-", "margin-left:%s;margin-right:%s"}, {"my-", "margin-top:%s;margin-bottom:%s"},
		{"mt-", "margin-top:%s"}, {"mr-", "margin-right:%s"}, {"mb-", "margin-bottom:%s"}, {"ml-", "margin-left:%s"}, {"m-", "margin:%s"},
	}
	for _, candidate := range prefixes {
		if !strings.HasPrefix(className, candidate.prefix) {
			continue
		}
		value := spacingValue(strings.TrimPrefix(className, candidate.prefix))
		if value == "" {
			return ""
		}
		return strings.ReplaceAll(candidate.properties, "%s", value)
	}
	return ""
}

func spacingValue(value string) string {
	values := map[string]string{
		"0": "0", "0.5": ".125rem", "1": ".25rem", "1.5": ".375rem", "2": ".5rem", "3": ".75rem", "4": "1rem", "5": "1.25rem", "6": "1.5rem", "7": "1.75rem", "8": "2rem", "9": "2.25rem", "10": "2.5rem", "12": "3rem", "14": "3.5rem", "16": "4rem", "20": "5rem", "24": "6rem", "32": "8rem",
	}
	return values[value]
}

func textSize(value string) string {
	values := map[string]string{
		"[13px]": "font-size:13px", "xs": "font-size:.75rem;line-height:1rem", "sm": "font-size:.875rem;line-height:1.25rem", "base": "font-size:1rem;line-height:1.5rem",
		"lg": "font-size:1.125rem;line-height:1.75rem", "xl": "font-size:1.25rem;line-height:1.75rem", "2xl": "font-size:1.5rem;line-height:2rem",
		"3xl": "font-size:1.875rem;line-height:2.25rem", "4xl": "font-size:2.25rem;line-height:2.5rem", "5xl": "font-size:3rem;line-height:1", "6xl": "font-size:3.75rem;line-height:1", "7xl": "font-size:4.5rem;line-height:1",
	}
	return values[value]
}

func colorValue(value string) string {
	colors := map[string]string{
		"transparent": "transparent", "white": "#fff", "black": "#000", "primary": "#d4f542", "primary-foreground": "#101918", "accent": "#f5ffd5",
		"slate-50": "#f8fafc", "slate-100": "#f1f5f9", "slate-200": "#e2e8f0", "slate-300": "#cbd5e1", "slate-400": "#94a3b8", "slate-500": "#64748b", "slate-600": "#475569", "slate-700": "#334155", "slate-800": "#1e293b", "slate-900": "#0f172a", "slate-950": "#020617",
		"gray-50": "#f9fafb", "gray-100": "#f3f4f6", "gray-200": "#e5e7eb", "gray-300": "#d1d5db", "gray-400": "#9ca3af", "gray-500": "#6b7280", "gray-600": "#4b5563", "gray-700": "#374151", "gray-800": "#1f2937", "gray-900": "#111827", "gray-950": "#030712",
		"zinc-50": "#fafafa", "zinc-100": "#f4f4f5", "zinc-200": "#e4e4e7", "zinc-300": "#d4d4d8", "zinc-400": "#a1a1aa", "zinc-500": "#71717a", "zinc-600": "#52525b", "zinc-700": "#3f3f46", "zinc-800": "#27272a", "zinc-900": "#18181b", "zinc-950": "#09090b",
		"stone-50": "#fafaf9", "stone-100": "#f5f5f4", "stone-200": "#e7e5e4", "stone-300": "#d6d3d1", "stone-400": "#a8a29e", "stone-500": "#78716c", "stone-600": "#57534e", "stone-700": "#44403c", "stone-800": "#292524", "stone-900": "#1c1917", "stone-950": "#0c0a09",
		"indigo-100": "#e0e7ff", "indigo-300": "#a5b4fc", "indigo-400": "#818cf8", "indigo-500": "#6366f1", "indigo-600": "#4f46e5",
		"amber-200": "#fde68a", "amber-300": "#fcd34d", "amber-400": "#fbbf24", "amber-500": "#f59e0b", "amber-950": "#451a03",
		"red-50": "#fef2f2", "red-100": "#fee2e2", "red-400": "#f87171", "red-500": "#ef4444", "red-600": "#dc2626", "red-700": "#b91c1c", "red-800": "#991b1b", "red-950": "#450a0a",
		"lime-100": "#ecfccb", "lime-200": "#d9f99d", "lime-300": "#bef264", "lime-950": "#1a2e05",
		"emerald-50": "#ecfdf5", "emerald-100": "#d1fae5", "emerald-400": "#34d399", "emerald-500": "#10b981", "emerald-600": "#059669", "emerald-700": "#047857",
		"rose-50": "#fff1f2", "rose-100": "#ffe4e6", "rose-400": "#fb7185", "rose-500": "#f43f5e", "rose-600": "#e11d48", "rose-700": "#be123c", "rose-950": "#4c0519",
		"sky-50": "#f0f9ff", "sky-100": "#e0f2fe", "sky-400": "#38bdf8", "sky-500": "#0ea5e9", "sky-600": "#0284c7", "sky-700": "#0369a1",
		"cyan-300": "#67e8f9", "cyan-400": "#22d3ee", "cyan-500": "#06b6d4",
	}
	if color := colors[value]; color != "" {
		return color
	}
	if arbitrary, ok := bracketValue(value); ok && strings.HasPrefix(arbitrary, "#") {
		return arbitrary
	}

	base, opacity, found := strings.Cut(value, "/")
	if !found {
		return ""
	}
	color := colors[base]
	if color == "" {
		if arbitrary, ok := bracketValue(base); ok && strings.HasPrefix(arbitrary, "#") {
			color = arbitrary
		}
	}
	if color == "" {
		return ""
	}
	alpha, ok := opacityValue(opacity)
	if !ok {
		return ""
	}
	return colorWithAlpha(color, alpha)
}

func opacityValue(value string) (float64, bool) {
	if bracketed, ok := bracketValue(value); ok {
		value = bracketed
	}
	amount, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	if amount > 1 {
		amount /= 100
	}
	if amount < 0 || amount > 1 {
		return 0, false
	}
	return amount, true
}

func colorWithAlpha(value string, alpha float64) string {
	hex := strings.TrimPrefix(value, "#")
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) != 6 {
		return value
	}
	red, redErr := strconv.ParseUint(hex[0:2], 16, 8)
	green, greenErr := strconv.ParseUint(hex[2:4], 16, 8)
	blue, blueErr := strconv.ParseUint(hex[4:6], 16, 8)
	if redErr != nil || greenErr != nil || blueErr != nil {
		return value
	}
	return "rgb(" + strconv.FormatUint(red, 10) + " " + strconv.FormatUint(green, 10) + " " + strconv.FormatUint(blue, 10) + " / " + strconv.FormatFloat(alpha, 'f', -1, 64) + ")"
}

func escapeClass(value string) string {
	var escaped strings.Builder
	for index, character := range value {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			character == '-' || character == '_' ||
			(character >= '0' && character <= '9' && index > 0) {
			escaped.WriteRune(character)
			continue
		}
		if character >= '0' && character <= '9' {
			escaped.WriteString(`\3`)
			escaped.WriteRune(character)
			escaped.WriteByte(' ')
			continue
		}
		escaped.WriteByte('\\')
		escaped.WriteRune(character)
	}
	return escaped.String()
}
