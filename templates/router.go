package templates

// Router is a template for the router file
var Router = parse("router",
	`{{ define "getParam" -}}
	{{- if eq . "path" -}}
		params.ByName
	{{- else if eq . "header" -}}
		r.Header.Get
	{{- else -}}
		query.Get
	{{- end -}}
{{ end -}}

{{/* Input: catch all error */}}
{{ define "unexpectedError" -}}
	respondJSON(w, m.errorTransformer.ErrorTo{{ if . }}{{ . }}{{ else }}String{{ end }}(err), "{{ if . }}{{ . }}{{ else }}string{{ end }}", http.StatusInternalServerError, errorTransformer)
{{ end -}}

package router

// This is a generated file
// Manual changes will be overwritten

import (
	"context"
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
	{{ .HandlerName }}(ctx context.Context,
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

// ErrorTransformer transforms errors in standard format into the format according to the swagger spec
type ErrorTransformer interface {
	{{ range .BadRequestErrors -}}
		ValidationErrorsTo{{ if eq "string" . }}String{{ else }}{{ . }}{{ end }}(errs []string) {{ if eq "string" . }}String{{ else }}model.{{ . }}{{ end }}
	{{ end -}}
	{{ range .InternalServerErrors -}}
		ErrorTo{{ if eq "string" . }}String{{ else }}{{ . }}{{ end }}(err error) {{ if eq "string" . }}string{{ else }}model.{{ . }}{{ end }}
	{{ end -}}

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

	return router
}

{{ range .Routes -}}
func (m *middleware) {{ .Name }}(w http.ResponseWriter, r *http.Request, {{ if .HasPathParams }}params{{ else }}_{{ end }} httprouter.Params) {
	errorTransformer := func(err error) interface{} { return m.errorTransformer.ErrorTo{{ if .CatchAllError }}{{ .CatchAllError }}{{ else }}String{{ end }}(err) }

	defer func() {
		if recovered := recover(); recovered != nil {
			err := errors.New("Recovered")
			log.WithField("error", recovered).Error(err)
			{{ template "unexpectedError" .CatchAllError -}}
		}
	}()

	var (
		{{ if .ResultType -}}
			result model.{{ if .ReadOnlyResult }}ReadOnly{{ end }}{{ .ResultType }}
		{{ end -}}
		err error
		{{ if or .ResultType .HasValidation -}}
			errs []string
		{{ end -}}
	)

	{{ if .HasQueryParams -}}
		query := r.URL.Query()
	{{ end -}}
	{{ range .Params -}}
		{{ if eq .Type "time.Time" -}}
			var {{ .Name }} time.Time
			if {{ .Name }}, err = time.Parse(time.RFC3339, {{ template "getParam" .Location }}("{{ .RawName }}")); err != nil {
				log.WithFields(log.Fields{
					"field": "{{ .RawName }}",
					"value": {{ template "getParam" .Location }}("{{ .RawName }}"),
				}).Error("Failed to parse time")
				errs = append(errs, "Failed to parse {{ .RawName }} as time")
			}
		{{ else if .IsArray -}}
			{{ .Name }} := parseArray({{ template "getParam" .Location }}("{{ .RawName }}"))
			{{ if .Validation.Array -}}
				errs = append(errs, validateArray({{ .Name }}, "{{ .RawName }}",
					{{- if .Validation.Array.HasMinItems -}} {{ .Validation.Array.MinItems }} {{- else -}} nil {{- end -}},
					{{- if .Validation.Array.HasMaxItems -}} {{ .Validation.Array.MaxItems }} {{- else -}} nil {{- end -}},
					{{- .Validation.Array.UniqueItems -}}
				)...)
			{{ end -}}
			{{ if .ItemValidation -}}
				for i := range {{ .Name }} {
					errs = append(errs, validateString({{ .Name }}[i], fmt.Sprintf("{{ .RawName }}[%d]", i),
						{{- if .ItemValidation.HasMinLength -}} {{ .ItemValidation.MinLength }} {{- else -}} nil {{- end -}},
						{{- if .ItemValidation.HasMaxLength -}} {{ .ItemValidation.MaxLength }} {{- else -}} nil {{- end -}},
						{{- if .ItemValidation.Enum -}} []string{ {{ .ItemValidation.FlattenedEnum }} } {{- else -}} nil {{- end -}}
					)...)
				}
			{{ end -}}
		{{ else -}}
			{{ .Name }} := {{ template "getParam" .Location }}("{{ .RawName }}")
			{{ if .Validation.String -}}
				errs = append(errs, validateString({{ .Name }}, "{{ .RawName }}",
					{{- if .Validation.String.HasMinLength -}} {{ .Validation.String.MinLength }} {{- else -}} nil {{- end -}},
					{{- if .Validation.String.HasMaxLength -}} {{ .Validation.String.MaxLength }} {{- else -}} nil {{- end -}},
					{{- if .Validation.String.Enum -}} []string{ {{ .Validation.String.FlattenedEnum }} } {{- else -}} nil {{- end -}}
				)...)
			{{ end -}}
		{{ end }}
	{{ end -}}

	{{ if .Body -}}
		var {{ .Body.Name }} model.{{ .Body.Type }}
		if err = json.NewDecoder(r.Body).Decode(&{{ .Body.Name }}); err != nil {
			errs = append(errs, err.Error())
			log.WithFields(log.Fields{
				"bodyType": "{{ .Body.Type }}",
				"error": err,
			}).Error("Failed to parse body data")
		} else if e := {{ .Body.Name }}.Validate(); len(e) > 0 {
			errs = append(errs, e...)
		}
	{{ end -}}

	{{ if .HasValidation -}}
		if len(errs) > 0 {
			log.WithFields(log.Fields{
				"handler": "{{ .Name }}",
				"errs": strings.Join(errs, "\n"),
			})
			respondJSON(w, m.errorTransformer.ValidationErrorsTo{{ if .ValidationError }}{{ .ValidationError }}{{ else }}String{{ end }}(errs), "{{ if .ValidationError }}{{ .ValidationError }}{{ else }}string{{ end }}", http.StatusBadRequest, errorTransformer)
			return
		}
	{{ end -}}

	if {{ if .ResultType }}result, {{ end }}err = m.handler.{{ .HandlerName }}(r.Context(),
		{{- range .Params -}}
			{{ .Name }},
		{{- end -}}
		{{- if .Body -}}
			{{ .Body.Name }}
		{{- end -}}
	); err != nil {
		switch err.(type) {
			{{ range .ResultErrors -}}
				case model.{{ .Type }}:
					respondJSON(w, err, "{{ .Type }}", {{ .StatusCode }}, errorTransformer)
			{{ end -}}
		default:
			{{ template "unexpectedError" .CatchAllError -}}
		}
		return
	}

	{{ if .ResultType -}}
		if errs = result.Validate(); len(errs) > 0 {
			err := errors.New("Invalid response data")
			log.WithFields(log.Fields{
				"dataType": "{{ if .ReadOnlyResult }}ReadOnly{{ end }}{{ .ResultType }}",
				"error": strings.Join(errs, "\n"),
			}).Error(err)
			{{ template "unexpectedError" .CatchAllError -}}
			return
		}

		respondJSON(w, result, "{{ if .ReadOnlyResult }}ReadOnly{{ end }}{{ .ResultType }}", http.StatusOK, errorTransformer)
	{{- else -}}
		w.Write([]byte("OK"))
	{{- end }}
}

{{ end -}}

func respondJSON(w http.ResponseWriter, data interface{}, dataType string, statusCode int, errorTransformer func(error) interface{}) {
	response, err := json.Marshal(data)
	if err != nil {
		log.WithFields(log.Fields{
			"dataType": dataType,
			"error": err.Error(),
		}).Error("Failed to convert to json")

		// we need to assume here that converting the error does not lead to json marshalling errors
		// it is the responsibility of the implementer to not mess this up
		response, _ = json.Marshal(errorTransformer(err))
		statusCode = http.StatusInternalServerError
	}

	// prevent caching
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate") // HTTP 1.1.
	w.Header().Set("Pragma", "no-cache") // HTTP 1.0.
	w.Header().Set("Expires", "0") // Proxies.

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(response)
}

func parseArray(s string) []string {
	// we treat the empty string as an empty array, rather than an array with one empty element
	if len(s) == 0 {
		return []string{}
	}
	return strings.Split(s, ",")
}

func validateString(s, name string, minLength, maxLength *int, enum []string) (errs []string) {
	if minLength != nil {
		if len(s) < *minLength {
			errs = append(errs, fmt.Sprintf("%s should be no shorter than %d characters", name, *minLength))
		}
	}

	if maxLength != nil {
		if len(s) > *maxLength {
			errs = append(errs, fmt.Sprintf("%s should be no longer than %d characters", name, *maxLength))
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
			errs = append(errs, fmt.Sprintf("%s is not an allowed value for %s", s, name))
		}
	}

	return
	}

	func validateArray(a []string, name string, minItems, maxItems *int, uniqueItems bool) (errs []string) {
	if minItems != nil {
		if len(a) < *minItems {
			errs = append(errs, fmt.Sprintf("%s should have no less than %d elements", name, *minItems))
		}
	}

	if maxItems != nil {
		if len(a) > *maxItems {
			errs = append(errs, fmt.Sprintf("%s should have no more than %d elements", name, *maxItems))
		}
	}

	if uniqueItems {
		seen := map[string]struct{}{}
		for _, elt := range a {
			if _, duplicate := seen[elt]; duplicate {
				errs = append(errs, fmt.Sprintf("%s occurs multiple times in %s", elt, name))
			}
			seen[elt] = struct{}{}
		}
	}

	return
}
`)
