// Package docs provides dependency-free primitives for documentation sites
// built with Northframe.
package docs

import (
	"fmt"
	"html"
	"io/fs"
	"path"
	"regexp"
	"strings"

	"github.com/JohnKinyanjui/northframe/pkg/web"
)

type Heading struct {
	Level int
	Title string
	ID    string
}

type Document struct {
	HTML     web.SafeHTML
	Headings []Heading
}

var (
	linkPattern   = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	strongPattern = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	codePattern   = regexp.MustCompile("`([^`]+)`")
)

// Load reads and renders a Markdown document from an application-owned fs.FS.
func Load(files fs.FS, name string) (Document, error) {
	clean := path.Clean(strings.TrimSpace(name))
	if clean == "." || strings.HasPrefix(clean, "../") || path.IsAbs(clean) {
		return Document{}, fmt.Errorf("invalid documentation path %q", name)
	}
	source, err := fs.ReadFile(files, clean)
	if err != nil {
		return Document{}, fmt.Errorf("read documentation %s: %w", clean, err)
	}
	return Render(source), nil
}

// Render converts Northframe's intentionally focused Markdown dialect into
// SafeHTML. Raw HTML is always escaped; links are limited to local anchors,
// application paths, and HTTPS URLs.
func Render(source []byte) Document {
	lines := strings.Split(strings.ReplaceAll(string(source), "\r\n", "\n"), "\n")
	var out strings.Builder
	var headings []Heading
	var paragraph []string
	inCode := false
	codeLanguage := ""
	var codeLines []string
	inList := false
	inQuote := false

	flushParagraph := func() {
		if len(paragraph) == 0 {
			return
		}
		out.WriteString(`<p class="mt-5 text-[1.02rem] leading-8 text-slate-700">` + renderInline(strings.Join(paragraph, " ")) + `</p>`)
		paragraph = nil
	}
	closeList := func() {
		if inList {
			out.WriteString("</ul>")
			inList = false
		}
	}
	closeQuote := func() {
		if inQuote {
			out.WriteString("</blockquote>")
			inQuote = false
		}
	}

	for _, raw := range lines {
		line := strings.TrimRight(raw, " \t")
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			flushParagraph()
			closeList()
			closeQuote()
			if inCode {
				out.WriteString(renderCodeBlock(codeLanguage, strings.Join(codeLines, "\n")))
				inCode = false
				codeLanguage = ""
				codeLines = nil
			} else {
				codeLanguage = strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
				inCode = true
			}
			continue
		}
		if inCode {
			codeLines = append(codeLines, line)
			continue
		}
		if trimmed == "" {
			flushParagraph()
			closeList()
			closeQuote()
			continue
		}
		if level, title, ok := heading(trimmed); ok {
			flushParagraph()
			closeList()
			closeQuote()
			id := slug(title)
			headings = append(headings, Heading{Level: level, Title: title, ID: id})
			classes := "scroll-mt-24 text-slate-950"
			switch level {
			case 1:
				classes += " hidden"
			case 2:
				classes += " mt-12 border-t border-slate-200 pt-10 text-2xl font-extrabold tracking-tight"
			default:
				classes += " mt-9 text-xl font-bold tracking-tight"
			}
			fmt.Fprintf(&out, `<h%d id="%s" class="%s"><a class="text-inherit no-underline" href="#%s">%s</a></h%d>`, level, id, classes, id, renderInline(title), level)
			continue
		}
		if strings.HasPrefix(trimmed, "- ") {
			flushParagraph()
			closeQuote()
			if !inList {
				out.WriteString(`<ul class="mt-5 space-y-2 pl-1">`)
				inList = true
			}
			out.WriteString(`<li class="flex gap-3 leading-7 text-slate-700"><span class="mt-3 h-1.5 w-1.5 shrink-0 rounded-full bg-orange-500"></span><span>` + renderInline(strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))) + "</span></li>")
			continue
		}
		if strings.HasPrefix(trimmed, "> ") {
			flushParagraph()
			closeList()
			if !inQuote {
				out.WriteString(`<blockquote class="mt-6 rounded-r-xl border-l-4 border-sky-500 bg-sky-50 px-5 pb-5 pt-0.5 text-sky-900">`)
				inQuote = true
			}
			out.WriteString(`<p class="mt-4 leading-7">` + renderInline(strings.TrimSpace(strings.TrimPrefix(trimmed, "> "))) + "</p>")
			continue
		}
		closeList()
		closeQuote()
		paragraph = append(paragraph, trimmed)
	}
	flushParagraph()
	closeList()
	closeQuote()
	if inCode {
		out.WriteString(renderCodeBlock(codeLanguage, strings.Join(codeLines, "\n")))
	}
	return Document{HTML: web.SafeHTMLFromSanitized(out.String()), Headings: headings}
}

func heading(line string) (int, string, bool) {
	for level := 1; level <= 4; level++ {
		prefix := strings.Repeat("#", level) + " "
		if strings.HasPrefix(line, prefix) {
			return level, strings.TrimSpace(strings.TrimPrefix(line, prefix)), true
		}
	}
	return 0, "", false
}

func renderInline(value string) string {
	escaped := html.EscapeString(value)
	escaped = linkPattern.ReplaceAllStringFunc(escaped, func(match string) string {
		parts := linkPattern.FindStringSubmatch(match)
		href := parts[2]
		if !strings.HasPrefix(href, "/") && !strings.HasPrefix(href, "#") && !strings.HasPrefix(href, "https://") {
			return parts[1] + " (" + href + ")"
		}
		return `<a class="font-semibold text-sky-700 underline decoration-sky-200 underline-offset-4 hover:text-sky-900" href="` + href + `">` + parts[1] + `</a>`
	})
	escaped = strongPattern.ReplaceAllString(escaped, `<strong class="font-bold text-slate-950">$1</strong>`)
	escaped = codePattern.ReplaceAllString(escaped, `<code class="rounded-md border border-slate-200 bg-slate-100 px-1.5 py-0.5 text-[0.9em] text-rose-700">$1</code>`)
	return escaped
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var out strings.Builder
	dash := false
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z', char >= '0' && char <= '9':
			out.WriteRune(char)
			dash = false
		case out.Len() > 0 && !dash:
			out.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(out.String(), "-")
}
