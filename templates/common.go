package templates

import (
	"errors"
	"strings"
	"text/template"
)

func parse(name, tpl string) *template.Template {
	return template.Must(template.New(name).Funcs(template.FuncMap{
		// allow on-the-fly maps in templates; see https://stackoverflow.com/a/18276968/2095090
		"dict": func(values ...interface{}) (dict map[string]interface{}, err error) {
			if len(values)%2 != 0 {
				err = errors.New("invalid dict call")
				return
			}
			dict = make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					err = errors.New("dict keys must be strings")
					return
				}
				dict[key] = values[i+1]
			}
			return
		},
		// replace ' by `: we never need ' and we are not allowed to use ` in templates...
	}).Parse(strings.Replace(tpl, "'", "`", -1)))
}
