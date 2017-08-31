package templates

// Model is a template for the model and errors file
var Model = parse("model",
	`{{/* Input: { Struct, ReadOnly } */ -}}
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
        {{ .Name }} {{ if .IsSlice }}[]{{ .ItemType }}{{ else }}*{{ .Type }}{{ end }} 'json:"{{ .JSONName }}" db:"{{ .JSONName }}"'
      {{ end -}}
    {{ end -}}
  }

  // New{{ .ReadOnly }}{{ .Struct.Name }} returns a new {{ .ReadOnly }}{{ .Struct.Name }}
  func New{{ .ReadOnly }}{{ .Struct.Name }}(
    {{- range .Struct.Props -}}
      {{ if or (eq $.ReadOnly "ReadOnly") (not .IsReadOnly) -}}
        {{ .JSONName }} {{ if .IsSlice }}[]{{ .ItemType }}{{ else }}{{ .Type }}{{ end }},
      {{- end -}}
    {{ end -}}
  ) {{ .ReadOnly }}{{ .Struct.Name }} {
    return {{ .ReadOnly }}{{ .Struct.Name }}{
      {{ if eq $.ReadOnly "ReadOnly" -}}
        New{{ .Struct.Name }}(
          {{- range .Struct.Props -}}
            {{ if not .IsReadOnly -}}
              {{ .JSONName }},
            {{- end -}}
          {{ end -}}
        ),
        {{ range .Struct.Props -}}
          {{ if .IsReadOnly -}}
            {{ if not .IsSlice }}&{{ end }}{{ .JSONName }},
          {{ end -}}
        {{ end -}}
      {{ else -}}
        {{ range .Struct.Props -}}
          {{ if not .IsReadOnly -}}
            {{ if not .IsSlice }}&{{ end }}{{ .JSONName }},
          {{ end -}}
        {{ end -}}
      {{ end -}}
    }
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
  {{ end -}}

  {{ if .IsError -}}
    func (e {{ .Name }}) Error() string {
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
      {{ else -}}
        return {{ .Type }}(e).Error()
      {{ end -}}
    }
  {{ end -}}
{{ end }}
`)
