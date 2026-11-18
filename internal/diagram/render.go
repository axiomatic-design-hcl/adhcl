package diagram

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"strings"
)

//go:embed assets/*.tmpl assets/page.css assets/page.js
var assets embed.FS

// The functions return template.HTML only for markup that they build from escaped
// text, so model text cannot inject markup.
var funcs = template.FuncMap{
	"wbr":    wbr,
	"ref":    renderRef,
	"refs":   func(rs []Ref) template.HTML { return joinRefs(rs, "<br>") },
	"inline": func(rs []Ref) template.HTML { return joinRefs(rs, ", ") },
	"tag":    tag,
	"gap": func(s string) template.HTML {
		return template.HTML(`<span class="none">` + string(tag(labelGap)) + " " + template.HTMLEscapeString(s) + `</span>`)
	},
	"none": func(s string) template.HTML {
		return template.HTML(`<span class="none">` + template.HTMLEscapeString(s) + `</span>`)
	},
	"cls": func(c string) template.HTML {
		return template.HTML(fmt.Sprintf(`<span class="cls %[1]s">%[1]s</span>`, template.HTMLEscapeString(c)))
	},
	"plural":   plural,
	"label":    func(name string) string { return labels[name] },
	"meaning":  func(code string) string { return codeGuides[code].Meaning },
	"junction": func(j string) string { return junctionMeaning[j] },
}

var labels = map[string]string{"source": labelSource, "inferred": labelInferred, "tool": labelTool, "derived": labelDerived, "gap": labelGap}

var pageTemplate = template.Must(template.New("page.html.tmpl").Funcs(funcs).ParseFS(assets, "assets/*.tmpl"))

// wbr escapes s and allows a line break after each dot and underscore, so long IDs
// break at their separators.
func wbr(s string) template.HTML {
	escaped := template.HTMLEscapeString(s)
	escaped = strings.ReplaceAll(escaped, ".", ".<wbr>")
	escaped = strings.ReplaceAll(escaped, "_", "_<wbr>")
	return template.HTML(escaped)
}

func renderRef(r Ref) template.HTML {
	var sb strings.Builder
	if r.Anchor {
		sb.WriteString(`<a class="ref" href="#` + template.HTMLEscapeString(r.ID) + `"><code>` + string(wbr(r.ID)) + `</code></a>`)
	} else {
		sb.WriteString(`<code>` + string(wbr(r.ID)) + `</code>`)
	}
	if r.Alias != "" {
		sb.WriteString(` <span class="alias">` + template.HTMLEscapeString(r.Alias) + `</span>`)
	}
	return template.HTML(sb.String())
}

func joinRefs(rs []Ref, sep string) template.HTML {
	parts := make([]string, 0, len(rs))
	for _, r := range rs {
		parts = append(parts, string(renderRef(r)))
	}
	return template.HTML(strings.Join(parts, sep))
}

var tagClass = map[string]string{labelSource: "source", labelInferred: "inferred", labelTool: "tool", labelDerived: "derived", labelGap: "gap"}

func tag(label string) template.HTML {
	class, ok := tagClass[label]
	if !ok {
		class = "source"
	}
	return template.HTML(`<span class="tag ` + class + `">` + template.HTMLEscapeString(label) + `</span>`)
}

// Render writes one self-contained page. Only the fixed, embedded assets are trusted.
func Render(w io.Writer, page Page) error {
	css, err := assets.ReadFile("assets/page.css")
	if err != nil {
		return err
	}
	js, err := assets.ReadFile("assets/page.js")
	if err != nil {
		return err
	}
	return pageTemplate.Execute(w, struct {
		Page
		CSS    template.CSS
		Script template.JS
	}{page, template.CSS(css), template.JS(js)})
}
