{{/* Input: { Struct, ReadOnly } */ -}}
{{ define "modelStruct" -}}
  // {{ .ReadOnly }}{{ .Struct.Name }}
    {{- if .Struct.Description }} {{ .Struct.Description }}{{ else }} No description provided{{ end }}
  type {{ .ReadOnly }}{{ .Struct.Name }} struct {
    {{ if .ReadOnly -}}
      {{ .Struct.Name }}
    {{ end -}}
    {{ range .Struct.Props -}}
      {{ if eq (eq $.ReadOnly "ReadOnly") .IsReadOnly -}}
        {{ if .Description -}}
          // {{ .Description }}
        {{ end -}}
        {{ .Name }} {{ if .IsSlice }}[]{{ .ItemType }}{{ else }}*{{ .Type }}{{ end }} `json:"{{ .JSONName }}" db:"{{ .JSONName }}"`
      {{ end -}}
    {{ end -}}
  }
{{ end -}}

{{/* Input: { Slice, ReadOnly } */ -}}
{{ define "modelSlice" }}
  // {{ .ReadOnly }}{{ .Slice.Name }}{{ if .Slice.Description }} {{ .Slice.Description }}{{ else }} No description provided{{ end }}
  type {{ .ReadOnly }}{{ .Slice.Name }} []{{ .ReadOnly }}{{ .Slice.ItemType }}
{{ end -}}

package model

// This is a generated file
// Manual changes will be overwritten

{{ range .Types -}}
  {{ if .IsStruct -}}
    {{ template "modelStruct" dict "Struct" . "ReadOnly" "" }}
    {{ if .HasReadOnlyProps -}}
      {{ template "modelStruct" dict "Struct" . "ReadOnly" "ReadOnly" }}
    {{ end -}}
  {{ else if .IsSlice -}}
    {{ template "modelSlice" dict "Slice" . "ReadOnly" "" }}
    {{ if .HasReadOnlyProps -}}
      {{ template "modelSlice" dict "Slice" . "ReadOnly" "ReadOnly" }}
    {{ end -}}
  {{ else -}}
    // {{ .Name }}{{ if .Description }} {{ .Description }}{{ else }} No description provided{{ end }}
    type {{ .Name }} {{ .Type }}
  {{ end }}
{{ end }}
