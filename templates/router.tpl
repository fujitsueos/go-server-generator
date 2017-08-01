{{ define "getParam" -}}
	{{- if eq . "path" -}}
		params.ByName
	{{- else if eq . "header" -}}
		r.Header.Get
	{{- else -}}
		query.Get
	{{- end -}}
{{ end -}}

package router

// This is a generated file
// Manual changes will be overwritten

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"

	"{{ .ModelPackage }}"
)

// Handler implements the actual functionality of the service
type Handler interface {
	{{ range .Routes -}}
		{{ if .NewBlock -}}
			// {{ .Tag }}
		{{ end -}}
		{{ .HandlerName }}(
			{{- range .Params -}}
				{{ .Name }} {{ if .IsArray }}[]{{ end }}{{ .Type }},
			{{- end -}}
			{{- if .Body -}}
				{{ .Body.Name }} model.{{ .Body.Type }}
			{{- end -}}
		) ({{ if .ResultType -}}
			model.{{ if .ReadOnlyResult }}ReadOnly{{ end }}{{ .ResultType }},
		{{- end }}error)
	{{ end }}
}

// ErrorTransformer transforms an error into a message and code that can be returned via http
type ErrorTransformer interface {
	Transform(err error) (message string, code int)
}

type middleware struct {
	handler Handler
	errorTransformer ErrorTransformer
}

// NewServer creates a http handler with a router for all methods of the service
func NewServer(handler Handler, errorTransformer ErrorTransformer) http.Handler {
	m := &middleware{
		handler,
		errorTransformer,
	}

	router := httprouter.New()

	{{ range .Routes -}}
		router.{{ .Method }}("{{ .Route }}", m.{{ .Name }})
	{{ end }}

	return Recoverer(router)
}

// Recoverer handles unexpected panics and returns internal server error
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.WithField("error", err).Error("Recovered")
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

{{ range .Routes -}}
	func (m *middleware) {{ .Name }}(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		var (
			data {{ .Name }}Data
			{{ if .ResultType -}}
				result model.{{ if .ReadOnlyResult }}ReadOnly{{ end }}{{ .ResultType }}
			{{ end -}}
			err error
			errors []string
		)

		if data, errors = parse{{ .HandlerName }}Data(r, params); len(errors) > 0 {
			log.WithFields(log.Fields{
				"handler": "{{ .Name }}",
				"errors": strings.Join(errors, "\n"),
			})
			http.Error(w, strings.Join(errors, "\n"), http.StatusBadRequest)
			return
		}

		if {{ if .ResultType }}result, {{ end }}err = m.handler.{{ .HandlerName }}(
			{{- range .Params -}}
				data.{{ .Name }},
			{{- end -}}
			{{- if .Body -}}
				data.{{ .Body.Name }}
			{{- end -}}
		); err != nil {
			message, code := m.errorTransformer.Transform(err)
			http.Error(w, message, code)
			return
		}

		{{ if .ResultType -}}
			if errors = result.Validate(); len(errors) > 0 {
				log.WithFields(log.Fields{
					"dataType": "{{ if .ReadOnlyResult }}ReadOnly{{ end }}{{ .ResultType }}",
					"error": strings.Join(errors, "\n"),
				}).Error("Invalid response data")
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			respondJSON(w, result, "{{ if .ReadOnlyResult }}ReadOnly{{ end }}{{ .ResultType }}")
		{{- else -}}
			w.Write([]byte("OK"))
		{{- end }}
	}

{{ end -}}

func respondJSON(w http.ResponseWriter, data interface{}, dataType string) {
	json, err := json.Marshal(data)
	if err != nil {
		log.WithFields(log.Fields{
			"dataType": dataType,
			"error": err.Error(),
		}).Error("Failed to convert to json")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

{{ range .Routes -}}
	type {{ .Name }}Data struct {
		{{- range .Params -}}
			{{ .Name }} {{ if .IsArray }}[]{{ end }}{{ .Type }}
		{{ end -}}
		{{ if .Body -}}
			{{ .Body.Name }} model.{{ .Body.Type }}
		{{ end -}}
	}

	func parse{{ .HandlerName }}Data(r *http.Request, {{ if .HasPathParams }}params{{ else }}_{{ end }} httprouter.Params) (data {{ .Name }}Data, errors []string) {
		{{ if .HasQueryParams -}}
			query := r.URL.Query()
		{{ end -}}
		{{ range .Params -}}
			{{ if eq .Type "time.Time" -}}
				if data.{{ .Name }}, err = time.Parse(time.RFC3339, {{ template "getParam" .Location }}("{{ .RawName }}")); err != nil {
					log.WithFields(log.Fields{
						"field": "{{ .RawName }}",
						"value": {{ template "getParam" .Location }}("{{ .RawName }}"),
					}).Error("Failed to parse time")
					return
				}
			{{ else if .IsArray -}}
				{{ .Name }} := {{ template "getParam" .Location }}("{{ .RawName }}")
				// we treat the empty string as an empty array, rather than an array with one empty element
				if len({{ .Name }}) == 0 {
					data.{{ .Name }} = []string{}
				} else {
					data.{{ .Name }} = strings.Split({{ .Name }}, ",")
				}
				{{ if .Validation.Array -}}
					errors = append(errors, validateArray(data.{{ .Name }}, "{{ .RawName }}",
						{{- if .Validation.Array.HasMinItems -}} {{ .Validation.Array.MinItems }} {{- else -}} nil {{- end -}},
						{{- if .Validation.Array.HasMaxItems -}} {{ .Validation.Array.MaxItems }} {{- else -}} nil {{- end -}},
						{{- .Validation.Array.UniqueItems -}}
					)...)
				{{ end -}}
				{{ if .ItemValidation -}}
					for i := range data.{{ .Name }} {
						errors = append(errors, validateString(data.{{ .Name }}[i], fmt.Sprintf("{{ .RawName }}[%d]", i),
							{{- if .ItemValidation.HasMinLength -}} {{ .ItemValidation.MinLength }} {{- else -}} nil {{- end -}},
							{{- if .ItemValidation.HasMaxLength -}} {{ .ItemValidation.MaxLength }} {{- else -}} nil {{- end -}},
							{{- if .ItemValidation.Enum -}} []string{ {{ .ItemValidation.FlattenedEnum }} } {{- else -}} nil {{- end -}}
						)...)
					}
				{{ end -}}
			{{ else -}}
				data.{{ .Name }} = {{ template "getParam" .Location }}("{{ .RawName }}")
				{{ if .Validation.String -}}
					errors = append(errors, validateString(data.{{ .Name }}, "{{ .RawName }}",
						{{- if .Validation.String.HasMinLength -}} {{ .Validation.String.MinLength }} {{- else -}} nil {{- end -}},
						{{- if .Validation.String.HasMaxLength -}} {{ .Validation.String.MaxLength }} {{- else -}} nil {{- end -}},
						{{- if .Validation.String.Enum -}} []string{ {{ .Validation.String.FlattenedEnum }} } {{- else -}} nil {{- end -}}
					)...)
				{{ end -}}
			{{ end }}
		{{ end -}}

		{{ if .Body -}}
			if err := json.NewDecoder(r.Body).Decode(&data.{{ .Body.Name }}); err != nil {
				errors = append(errors, err.Error())
				log.WithFields(log.Fields{
					"bodyType": "{{ .Body.Type }}",
					"error": err,
				}).Error("Failed to parse body data")
				return
			}
			if e := data.{{ .Body.Name }}.Validate(); len(e) > 0 {
				errors = append(errors, e...)
			}
		{{ end -}}

		return
	}
{{ end -}}

func validateString(s, name string, minLength, maxLength *int, enum []string) (errors []string) {
	if minLength != nil {
		if len(s) < *minLength {
			errors = append(errors, fmt.Sprintf("%s should be no shorter than %d characters", name, *minLength))
		}
	}

	if maxLength != nil {
		if len(s) > *maxLength {
			errors = append(errors, fmt.Sprintf("%s should be no longer than %d characters", name, *maxLength))
		}
	}

	if enum != nil {
		found := false
		for i := range enum {
			if s == enum[i] {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, fmt.Sprintf("%s is not an allowed value for %s", s, name))
		}
	}

	return
}

func validateArray(a []string, name string, minItems, maxItems *int, uniqueItems bool) (errors []string) {
	if minItems != nil {
		if len(a) < *minItems {
			errors = append(errors, fmt.Sprintf("%s should have no less than %d elements", name, *minItems))
		}
	}

	if maxItems != nil {
		if len(a) > *maxItems {
			errors = append(errors, fmt.Sprintf("%s should have no more than %d elements", name, *maxItems))
		}
	}

	if uniqueItems {
		seen := map[string]struct{}{}
		for _, elt := range a {
			if _, duplicate := seen[elt]; duplicate {
				errors = append(errors, fmt.Sprintf("%s occurs multiple times in %s", elt, name))
			}
			seen[elt] = struct{}{}
		}
	}

	return
}
