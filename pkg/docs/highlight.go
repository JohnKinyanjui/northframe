package docs

import (
	"html"
	"strings"
	"unicode"
)

var syntaxKeywords = map[string]bool{
	"break": true, "case": true, "const": true, "continue": true, "default": true,
	"defer": true, "else": true, "fallthrough": true, "for": true, "func": true,
	"go": true, "goto": true, "if": true, "import": true, "interface": true,
	"map": true, "package": true, "range": true, "return": true, "select": true,
	"struct": true, "switch": true, "type": true, "var": true,
	"async": true, "await": true, "class": true, "export": true, "extends": true,
	"function": true, "implements": true, "let": true, "new": true, "private": true,
	"public": true, "readonly": true, "throw": true, "try": true, "catch": true,
	"from": true, "as": true, "in": true, "of": true,
	"and": true, "or": true, "not": true, "null": true,
	"create": true, "table": true, "insert": true, "into": true, "values": true,
	"update": true, "delete": true, "where": true, "join": true, "on": true,
	"order": true, "by": true, "group": true, "limit": true,
}

var syntaxTypes = map[string]bool{
	"any": true, "bool": true, "byte": true, "error": true, "float32": true,
	"float64": true, "int": true, "int8": true, "int16": true, "int32": true,
	"int64": true, "rune": true, "string": true, "uint": true, "uint8": true,
	"uint16": true, "uint32": true, "uint64": true, "uintptr": true, "void": true,
	"boolean": true, "number": true, "unknown": true, "Props": true,
}

func renderCodeBlock(language, source string) string {
	language = strings.ToLower(strings.TrimSpace(language))
	label := language
	if label == "" {
		label = "text"
	}
	if (language == "html" || language == "north") && looksLikeNorth(source) {
		label = "north"
	}
	var highlighted string
	switch language {
	case "html", "north":
		highlighted = highlightNorth(source)
	case "go", "golang", "ts", "typescript", "js", "javascript", "sql", "toml", "sh", "shell", "bash":
		highlighted = highlightProgram(source, language)
	default:
		highlighted = html.EscapeString(source)
	}
	return `<pre class="relative mt-6 overflow-x-auto rounded-xl border border-slate-800 bg-slate-950 px-5 pb-5 pt-11 text-sm leading-7 text-slate-100 shadow-sm" data-language="` + html.EscapeString(label) + `"><span class="absolute right-4 top-3 select-none text-[10px] font-bold uppercase tracking-[0.18em] text-slate-500">` + html.EscapeString(label) + `</span><code class="language-` + html.EscapeString(label) + `">` + highlighted + `</code></pre>`
}

func looksLikeNorth(source string) bool {
	return strings.Contains(source, "${") || strings.Contains(source, "#{") ||
		strings.Contains(source, "interface Props") || strings.Contains(source, "{for ") ||
		strings.Contains(source, "{if ") || strings.Contains(source, "<slot")
}

func highlightNorth(source string) string {
	var out strings.Builder
	for index := 0; index < len(source); {
		if strings.HasPrefix(source[index:], "<!--") {
			end := strings.Index(source[index+4:], "-->")
			if end < 0 {
				end = len(source) - index - 4
			} else {
				end += 3
			}
			writeToken(&out, "text-slate-500", source[index:index+4+end])
			index += 4 + end
			continue
		}
		if source[index] == '{' {
			if end := strings.IndexByte(source[index:], '}'); end >= 0 {
				expression := source[index : index+end+1]
				writeToken(&out, "text-orange-300", expression[:1])
				out.WriteString(highlightProgram(expression[1:len(expression)-1], "go"))
				writeToken(&out, "text-orange-300", expression[len(expression)-1:])
				index += end + 1
				continue
			}
		}
		if source[index] == '<' {
			if end := strings.IndexByte(source[index:], '>'); end >= 0 {
				out.WriteString(highlightTag(source[index : index+end+1]))
				index += end + 1
				continue
			}
		}
		start := index
		for index < len(source) && source[index] != '<' && source[index] != '{' {
			index++
		}
		out.WriteString(html.EscapeString(source[start:index]))
	}
	return out.String()
}

func highlightTag(tag string) string {
	var out strings.Builder
	for index := 0; index < len(tag); {
		char := tag[index]
		switch {
		case char == '<' || char == '>' || char == '/' || char == '=':
			writeToken(&out, "text-slate-500", tag[index:index+1])
			index++
		case char == '"' || char == '\'':
			end := scanQuoted(tag, index)
			writeToken(&out, "text-emerald-300", tag[index:end])
			index = end
		case isIdentifierStart(rune(char)):
			start := index
			for index < len(tag) && (isIdentifierPart(rune(tag[index])) || tag[index] == ':' || tag[index] == '-') {
				index++
			}
			class := "text-sky-300"
			if start > 0 && (tag[start-1] == '<' || tag[start-1] == '/') {
				class = "font-semibold text-rose-300"
			}
			writeToken(&out, class, tag[start:index])
		default:
			out.WriteString(html.EscapeString(tag[index : index+1]))
			index++
		}
	}
	return out.String()
}

func highlightProgram(source, language string) string {
	var out strings.Builder
	for index := 0; index < len(source); {
		char := source[index]
		switch {
		case strings.HasPrefix(source[index:], "//") || strings.HasPrefix(source[index:], "--"):
			end := strings.IndexByte(source[index:], '\n')
			if end < 0 {
				end = len(source) - index
			}
			writeToken(&out, "text-slate-500", source[index:index+end])
			index += end
		case strings.HasPrefix(source[index:], "/*"):
			end := strings.Index(source[index+2:], "*/")
			if end < 0 {
				end = len(source) - index - 2
			} else {
				end += 2
			}
			writeToken(&out, "text-slate-500", source[index:index+2+end])
			index += 2 + end
		case char == '#' && (language == "sh" || language == "shell" || language == "bash" || language == "toml"):
			end := strings.IndexByte(source[index:], '\n')
			if end < 0 {
				end = len(source) - index
			}
			writeToken(&out, "text-slate-500", source[index:index+end])
			index += end
		case char == '"' || char == '\'' || char == '`':
			end := scanQuoted(source, index)
			writeToken(&out, "text-emerald-300", source[index:end])
			index = end
		case char >= '0' && char <= '9':
			start := index
			for index < len(source) && ((source[index] >= '0' && source[index] <= '9') || source[index] == '.') {
				index++
			}
			writeToken(&out, "text-amber-300", source[start:index])
		case isIdentifierStart(rune(char)):
			start := index
			for index < len(source) && isIdentifierPart(rune(source[index])) {
				index++
			}
			word := source[start:index]
			lower := strings.ToLower(word)
			switch {
			case syntaxKeywords[word] || syntaxKeywords[lower]:
				writeToken(&out, "font-semibold text-rose-300", word)
			case syntaxTypes[word] || syntaxTypes[lower] || unicode.IsUpper(rune(word[0])):
				writeToken(&out, "text-sky-300", word)
			case lower == "true" || lower == "false" || lower == "nil":
				writeToken(&out, "text-violet-300", word)
			default:
				out.WriteString(html.EscapeString(word))
			}
		default:
			out.WriteString(html.EscapeString(source[index : index+1]))
			index++
		}
	}
	return out.String()
}

func scanQuoted(source string, start int) int {
	quote := source[start]
	for index := start + 1; index < len(source); index++ {
		if source[index] == '\\' && quote != '`' {
			index++
			continue
		}
		if source[index] == quote {
			return index + 1
		}
	}
	return len(source)
}

func isIdentifierStart(char rune) bool {
	return char == '_' || char == '$' || unicode.IsLetter(char)
}

func isIdentifierPart(char rune) bool {
	return isIdentifierStart(char) || unicode.IsDigit(char) || char == '.'
}

func writeToken(out *strings.Builder, class, value string) {
	out.WriteString(`<span class="` + class + `">` + html.EscapeString(value) + `</span>`)
}
