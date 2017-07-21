package model

// This is a generated file
// Manual changes will be overwritten

{{ if .NeedsTime -}}
import "time"

{{ end -}}

{{ range .Types -}}
{{ if eq .Type "struct" -}}
// {{ .Name }}{{ if .Description }} {{ .Description }}{{ else }} No description provided{{ end }}
type {{ .Name }} struct {
{{ range .Props -}}
{{ if .Description }}	// {{ .Description }}
{{ end -}}
{{ "	" }}{{ .Name }} *{{ .Type }} `json:"{{ .JSONName }}"`
{{ end -}}
}
{{ else -}}
// {{ .Name }}{{ if .Description }} {{ .Description }}{{ else }} No description provided{{ end }}
type {{ .Name }} []{{ .RefType }}
{{ end }}
{{ end }}
