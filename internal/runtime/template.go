package runtime

import (
    "bytes"
    "text/template"
)

func RenderTemplate(input string, vars map[string]string) (string, error) {
    tmpl, err := template.New("tmpl").Parse(input)
    if err != nil {
        return "", err
    }
    var buf bytes.Buffer
    err = tmpl.Execute(&buf, vars)
    return buf.String(), err
}
