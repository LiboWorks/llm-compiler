package runtime

import (
	"bytes"
	"regexp"
	"text/template"
)

// RenderTemplate renders a user-provided template using vars as the map of
// values. To be more user-friendly we allow the shorthand {{key}} (no dot)
// and rewrite it to use `index` so it works with map[string]string data.
var bareVarRe = regexp.MustCompile(`{{\s*([a-zA-Z0-9_]+)\s*}}`)

func RenderTemplate(input string, vars map[string]string) (string, error) {
	// rewrite occurrences of {{key}} -> {{ index . "key" }} so templates
	// written by users (e.g. {{lang}}) work against a map[string]string.
	rewritten := bareVarRe.ReplaceAllString(input, `{{ index . "$1" }}`)

	tmpl, err := template.New("tmpl").Parse(rewritten)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, vars)
	return buf.String(), err
}
