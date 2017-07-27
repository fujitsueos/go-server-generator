package model

// This is a generated file
// Manual changes will be overwritten

{{ if .NeedsTime -}}
import "time"

{{ end -}}

{{ range .Types -}}
  // {{ .Name }}{{ if .Description }} {{ .Description }}{{ else }} No description provided{{ end }}
  {{ if .IsStruct -}}
    type {{ .Name }} struct {
      {{ range .Props -}}
        {{ if .Description -}}
          // {{ .Description }}
        {{ end -}}
        {{ .Name }} {{ if .IsSlice }}[]{{ .ItemType }}{{ else }}*{{ .Type }}{{ end }} `json:"{{ .JSONName }}"`
      {{ end -}}
    }
  {{ else if .IsSlice -}}
    type {{ .Name }} []{{ .ItemType }}
  {{ else -}}
    type {{ .Name }} {{ .Type }}
  {{ end }}
{{ end }}
