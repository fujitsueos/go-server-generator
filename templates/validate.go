package templates

// Validate is a template for the validate file
var Validate = parse("validate", `
	{{/* Input: { Type, ReadOnly } */}}
	{{ define "validateType" -}}
		// Validate validates a {{ .ReadOnly }}{{ .Type.Name }} based on the swagger spec
		func (s *{{ .ReadOnly }}{{ .Type.Name }}) Validate() (errors []string) {
			{{ if .Type.IsStruct -}}
				{{ if .ReadOnly -}}
					if e := s.{{ .Type.Name }}.Validate(); len(e) > 0 {
						errors = append(errors, e...)
					}
				{{ end -}}

				{{ range .Type.Props -}}
					{{ if eq (eq $.ReadOnly "ReadOnly") .IsReadOnly -}}
						{{ if .IsRequired }}
							if s.{{ .Name }} == nil {
								errors = append(errors, "{{ .JSONName }} is required")
							}

							{{- if .IsSlice -}}
								{{ $else := templateAsString "validateSlice" (dict "Validation" .Validation.Array "Slice" (print "s." .Name) "Name" .JSONName "ItemType" .ItemType "ItemValidation" .ItemValidation "RegexpName" (print $.Type.Name .Name)) -}}
								{{- if $else -}}
									else {
										{{ $else }}
									}
								{{ end -}}
							{{- else if eq .Type "int64" -}}
								{{ $else := templateAsString "validateInt64" (dict "Validation" .Validation.Int "Int" (print "*s." .Name) "Name" .JSONName) -}}
								{{- if $else -}}
									else {
										{{ $else }}
									}
								{{ end -}}
							{{- else if eq .Type "float64" -}}
								{{ $else := templateAsString "validateFloat64" (dict "Validation" .Validation.Number "Number" (print "*s." .Name) "Name" .JSONName) -}}
								{{- if $else -}}
									else {
										{{ $else }}
									}
								{{ end -}}
							{{- else if eq .Type "string" -}}
								{{ $else := templateAsString "validateString" (dict "Validation" .Validation.String "String" (print "*s." .Name) "Name" .JSONName "RegexpName" (print $.Type.Name .Name)) -}}
								{{- if $else -}}
									else {
										{{ $else }}
									}
								{{ end -}}
							{{- else if not (eq .Type "bool" "time.Time") -}}
								else {
									if e := s.{{ .Name }}.Validate(); len(e) > 0 {
										errors = append(errors, e...)
									}
								}
							{{ end -}}
						{{ end -}}
					{{ end -}}
				{{ end }}
			{{ else }}{{/* .Type.IsSlice */ -}}
				{{ template "validateSlice" dict "Validation" .Type.Validation.Array "Slice" "*s" "Name" .Type.Name "ItemType" (print .ReadOnly .Type.ItemType) "ItemValidation" .Type.ItemValidation "RegexpName" $.Type.Name -}}
			{{ end -}}

			return
		}
	{{ end -}}

	{{/* Input: { Slice, Name, Validation, ItemType, ItemValidation, RegexpName } */ -}}
	{{ define "validateSlice" -}}
		{{ if .Validation -}}
			{{ if .Validation.HasMaxItems }}
				if len({{ .Slice }}) > {{ .Validation.MaxItems }} {
					errors = append(errors, "{{ .Name }} should have no more than {{ .Validation.MaxItems }} elements")
				}
			{{ end -}}

			{{ if .Validation.HasMinItems }}
				if len({{ .Slice }}) < {{ .Validation.MinItems }} {
					errors = append(errors, "{{ .Name }} should have no less than {{ .Validation.MinItems }} elements")
				}
			{{ end -}}

			{{ if .Validation.UniqueItems -}}
				unique := make(map[{{ .ItemType }}]struct{})
				for _, elt := range {{ .Slice }} {
					unique[elt] = struct{}{}
				}
				if len(unique) < len({{ .Slice }}) {
					errors = append(errors, "{{ .Name }} contains duplicate elements")
				}
			{{ end -}}
		{{ end -}}

		{{ if eq .ItemType "int64" -}}
			{{ if .ItemValidation.Int }}
				for i, elt := range {{ .Slice }} {
					{{- template "validateInt64" dict "Validation" .ItemValidation.Int "Int" "elt" "Name" (print .Name "[%d]") "FormatParams" "i" -}}
				}
			{{ end -}}
		{{ else if eq .ItemType "float64" -}}
			{{ if .ItemValidation.Number }}
				for i, elt := range {{ .Slice }} {
					{{- template "validateFloat64" dict "Validation" .ItemValidation.Number "Number" "elt" "Name" (print .Name "[%d]") "FormatParams" "i" -}}
				}
			{{ end -}}
		{{ else if eq .ItemType "string" -}}
			{{ if .ItemValidation.String }}
				for i, elt := range {{ .Slice }} {
					{{- template "validateString" dict "Validation" .ItemValidation.String "String" "elt" "Name" (print .Name "[%d]") "RegexpName" $.RegexpName "FormatParams" "i" -}}
				}
			{{ end -}}
		{{ else if not (eq .ItemType "bool" "time.Time") }}
			for _, elt := range {{ .Slice }} {
				if e := elt.Validate(); len(e) > 0 {
					errors = append(errors, e...)
				}
			}
		{{ end -}}
	{{ end -}}

	{{/* Input: { Int, Name, Validation, FormatParams (optional) } */ -}}
	{{ define "validateInt64" -}}
		{{ if .Validation -}}
			{{ if .Validation.Enum }}
				switch {{ .Int }} {
				case {{ .Validation.FlattenedEnum }}: // ok
				default:
					errors = append(errors, fmt.Sprintf("%d is not an allowed value for {{ .Name }}", {{ .Int }}
						{{- if .FormatParams }}, {{ .FormatParams }}{{ end }}))
				}
			{{ end -}}

			{{- if .Validation.HasMaximum }}
				if {{ .Int }} {{ if .Validation.ExclusiveMaximum }}>={{ else }}>{{ end }} {{ .Validation.Maximum }} {
					errors = append(errors,
						{{- if .FormatParams }}fmt.Sprintf({{ end -}}
						"{{ .Name }} should be {{ if .Validation.ExclusiveMaximum }}less than{{ else }}at most{{ end }} {{ .Validation.Maximum }}"
						{{- if .FormatParams }}, {{ .FormatParams }}){{ end }})
				}
			{{ end -}}

			{{- if .Validation.HasMinimum }}
				if {{ .Int }} {{ if .Validation.ExclusiveMinimum }}<={{ else }}<{{ end }} {{ .Validation.Minimum }} {
					errors = append(errors,
						{{- if .FormatParams }}fmt.Sprintf({{ end -}}
						"{{ .Name }} should be {{ if .Validation.ExclusiveMinimum }}more than{{ else }}at least{{ end }} {{ .Validation.Minimum }}"
						{{- if .FormatParams }}, {{ .FormatParams }}){{ end }})
				}
			{{ end -}}
		{{ end -}}
	{{ end -}}

	{{/* Input: { Number, Name, Validation, FormatParams (optional) } */ -}}
	{{ define "validateFloat64" -}}
		{{ if .Validation -}}
			{{ if .Validation.Enum }}
				switch {{ .Number }} {
				case {{ .Validation.FlattenedEnum }}: // ok
				default:
					errors = append(errors, fmt.Sprintf("%f is not an allowed value for {{ .Name }}", {{ .Number }}
						{{- if .FormatParams }}, {{ .FormatParams }}{{ end }}))
				}
			{{ end -}}

			{{- if .Validation.HasMaximum }}
				if {{ .Number }} {{ if .Validation.ExclusiveMaximum }}>={{ else }}>{{ end }} {{ .Validation.Maximum }} {
					errors = append(errors,
						{{- if .FormatParams }}fmt.Sprintf({{ end -}}
						"{{ .Name }} should be {{ if .Validation.ExclusiveMaximum }}less than{{ else }}at most{{ end }} {{ .Validation.Maximum }}"
						{{- if .FormatParams }}, {{ .FormatParams }}){{ end }})
				}
			{{ end -}}

			{{- if .Validation.HasMinimum }}
				if {{ .Number }} {{ if .Validation.ExclusiveMinimum }}<={{ else }}<{{ end }} {{ .Validation.Minimum }} {
					errors = append(errors,
						{{- if .FormatParams }}fmt.Sprintf({{ end -}}
						"{{ .Name }} should be {{ if .Validation.ExclusiveMinimum }}more than{{ else }}at least{{ end }} {{ .Validation.Minimum }}"
						{{- if .FormatParams }}, {{ .FormatParams }}){{ end }})
				}
			{{ end -}}
		{{ end -}}
	{{ end -}}

	{{/* Input: { String, Name, Validation, RegexpName, FormatParams (optional) } */ -}}
	{{ define "validateString" -}}
		{{ if .Validation -}}
			{{ if .Validation.Enum }}
				switch {{ .String }} {
				case {{ .Validation.FlattenedEnum }}: // ok
				default:
					errors = append(errors, fmt.Sprintf("%s is not an allowed value for {{ .Name }}", {{ .String }}
						{{- if .FormatParams }}, {{ .FormatParams }}{{ end }}))
				}
			{{ end -}}

			{{- if .Validation.HasMaxLength }}
				if len({{ .String }}) > {{ .Validation.MaxLength }} {
					errors = append(errors,
						{{- if .FormatParams }}fmt.Sprintf({{ end -}}
						"{{ .Name }} should be no longer than {{ .Validation.MaxLength }} characters"
						{{- if .FormatParams }}, {{ .FormatParams }}){{ end }})
				}
			{{ end -}}

			{{- if .Validation.HasMinLength }}
				if len({{ .String }}) < {{ .Validation.MinLength }} {
					errors = append(errors,
						{{- if .FormatParams }}fmt.Sprintf({{ end -}}
						"{{ .Name }} should be no shorter than {{ .Validation.MinLength }} characters"
						{{- if .FormatParams }}, {{ .FormatParams }}){{ end }})
				}
			{{ end -}}

			{{- if .Validation.HasPattern }}
				if !regexp{{ $.RegexpName }}.MatchString({{ .String }}) {
					errors = append(errors,
						{{- if .FormatParams }}fmt.Sprintf({{ end -}}
						"{{ .Name }} should match the regex {{ .Validation.Pattern }}"
						{{- if .FormatParams }}, {{ .FormatParams }}){{ end }})
				}
			{{ end -}}
		{{ end -}}
	{{ end -}}

	package model

	// This is a generated file
	// Manual changes will be overwritten

	{{ range .Types -}}
		{{ if or .IsStruct .IsSlice -}}
			{{ template "validateType" dict "Type" . "ReadOnly" "" }}
			{{ if .HasReadOnlyProps -}}
				{{ template "validateType" dict "Type" . "ReadOnly" "ReadOnly" }}
			{{ end -}}
		{{ else -}}
			// Validate validates a {{ .Name }} based on the swagger spec
			func (s *{{ .Name }}) Validate() (errors []string) {
				{{ if eq .Type "int64" -}}
					{{ template "validateInt64" dict "Validation" .Validation.Int "Int" "int64(*s)" "Name" .Name -}}
				{{ else if eq .Type "float64" -}}
					{{ template "validateFloat64" dict "Validation" .Validation.Number "Number" "float64(*s)" "Name" .Name -}}
				{{ else if eq .Type "string" -}}
					{{ template "validateString" dict "Validation" .Validation.String "String" "string(*s)" "Name" .Name "RegexpName" .Name -}}
				{{ end }}

				return
			}
		{{ end -}}
	{{ end }}

	{{ range .Patterns -}}
		var regexp{{ .Name }} = regexp.MustCompile(`+"`"+`{{ .Pattern }}`+"`"+`)
	{{ end -}}
`)
