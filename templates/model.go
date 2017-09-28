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

  {{/* Note that (and .ReadOnly .Struct.Name) means (if .ReadOnly == "" then "" else .Struct.Name) */}}
  {{ template "constructor" dict "Name" (printf "%s%s" .ReadOnly .Struct.Name) "NonReadOnlyName" (and .ReadOnly .Struct.Name) "Props" .Struct.Props }}
{{ end -}}

{{/* Input: { Slice, ReadOnly } */ -}}
{{ define "modelSlice" }}
  // {{ .ReadOnly }}{{ .Slice.Name }}{{ if .Slice.Description }} {{ .Slice.Description }}{{ else }} No description provided{{ end }}
  type {{ .ReadOnly }}{{ .Slice.Name }} []{{ .ReadOnly }}{{ .Slice.ItemType }}
{{ end -}}

{{/* Input: { Name, NonReadOnlyName, ReferenceName, Props } */ -}}
{{ define "constructor" }}
  // New{{ .Name }} returns a new {{ .Name }}
  func New{{ .Name }}(
    {{- range .Props -}}
      {{ if or $.NonReadOnlyName (not .IsReadOnly) -}}
        {{ .JSONName }} {{ if .IsSlice }}[]{{ .ItemType }}{{ else }}{{ .Type }}{{ end }},
      {{- end -}}
    {{ end -}}
  ) {{ .Name }} {
    {{ if .ReferenceName -}}
      return {{ .Name }}(New{{ .ReferenceName }}(
        {{- range .Props -}}
          {{ .JSONName }},
        {{- end -}}
      ))
    {{ else -}}
      return {{ .Name }}{
        {{ if $.NonReadOnlyName -}}
          New{{ .NonReadOnlyName }}(
            {{- range .Props -}}
              {{ if not .IsReadOnly -}}
                {{ .JSONName }},
              {{- end -}}
            {{ end -}}
          ),
          {{ range .Props -}}
            {{ if .IsReadOnly -}}
              {{ if not .IsSlice }}&{{ end }}{{ .JSONName }},
            {{ end -}}
          {{ end -}}
        {{ else -}}
          {{ range .Props -}}
            {{ if not .IsReadOnly -}}
              {{ if not .IsSlice }}&{{ end }}{{ .JSONName }},
            {{ end -}}
          {{ end -}}
        {{ end -}}
      }
    {{ end -}}
  }
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

    {{ if .Ref -}}
      {{ template "constructor" dict "Name" .Name "ReferenceName" .Ref.Name "Props" .Ref.Props }}
    {{ end -}}
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
      {{ else if eq .Type "string" -}}
        return string(e)
      {{ else -}}
        return {{ .Type }}(e).Error()
      {{ end -}}
    }
  {{ end -}}
{{ end }}
`)
