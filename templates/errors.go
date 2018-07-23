package templates

// Errors is a template for the errors file
var Errors = parse("model",
	`package model

// This is a generated file
// Manual changes will be overwritten

{{ range .BaseErrors -}}
	type {{ . }} interface{}
{{ end }}

{{ range .Types -}}
	{{ if .IsStruct -}}
		type {{ .PrivateName }}Impl struct {
			{{ range .Props -}}
				{{ if .Description -}}
					// {{ .Description }}
				{{ end -}}
				{{ .Name }} {{ if .IsSlice }}[]{{ .ItemType }}{{ else }}*{{ .Type }}{{ end }} 'json:"{{ .JSONName }}" db:"{{ .JSONName }}"'
			{{ end -}}
		}
  {{ else if .Ref -}}
		type {{ .PrivateName }}Impl {{ .Ref.PrivateName }}Impl
	{{ else -}}
		type {{ .PrivateName }}Impl string
	{{ end -}}

	func (e *{{ .PrivateName }}Impl) Error() string {
		{{ if or .IsStruct .IsSlice -}}
			return spew.Sdump(struct{
				{{ range .Props -}}
					{{ .Name }} {{ if .IsSlice }}[]{{ .ItemType }}{{ else }}*{{ .Type }}{{ end }}
				{{ end -}}
			}{
				{{- range .Props -}}
					e.{{ .Name }},
				{{- end -}}
			})
		{{ else if .Ref -}}
			return (*{{ .Ref.PrivateName }}Impl)(e).Error()
		{{ else -}}
			return string(*e)
		{{ end -}}
	}

	{{ if .IsStruct -}}
		func New{{ .Name }}(
			{{- range .Props -}}
        {{ .JSONName }} {{ if .IsSlice }}[]{{ .ItemType }}{{ else }}{{ .Type }}{{ end }},
    	{{- end -}}
		) {{ .Name }} {
			return &{{ .PrivateName }}Impl{
				{{- range .Props -}}
					{{ .Name }}: {{ if not .IsSlice }}&{{ end }}{{ .JSONName }},
    		{{ end -}}
			}
		}
	{{ else if .Ref -}}
		func New{{ .Name }}(
			{{- range .Ref.Props -}}
				{{ .JSONName }} {{ if .IsSlice }}[]{{ .ItemType }}{{ else }}{{ .Type }}{{ end }},
			{{- end -}}
		) {{ .Name }} {
			base, _ := New{{ .Ref.Name }}(
				{{- range .Ref.Props -}}
					{{ .JSONName }},
				{{ end -}}
			).(*{{ .Ref.PrivateName }}Impl)
			return (*{{ .PrivateName }}Impl)(base)
		}
	{{ else -}}
		func New{{ .Name }}(s string) {{ .Name }} {
			return (*{{ .JSONName }})(&s)
		}
	{{ end -}}
{{ end }}
`)
