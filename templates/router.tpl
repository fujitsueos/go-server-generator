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
	func (m *middleware) {{ .Name }}(w http.ResponseWriter, r *http.Request, {{ if .HasPathParams }}params{{ else }}_{{ end }} httprouter.Params) {
		{{ if .HasQueryParams -}}
			query := r.URL.Query()
		{{ end -}}
		{{- range .Params -}}
			{{ if eq .Type "time.Time" -}}
				{{ .Name }}, parseErr := time.Parse(time.RFC3339, {{ template "getParam" .Location }}("{{ .RawName }}"))
				if parseErr != nil {
					http.Error(w, "Cannot parse date", http.StatusBadRequest)
					return
				}
			{{ else if .IsArray -}}
				{{ .Name }} := strings.Split({{ template "getParam" .Location }}("{{ .RawName }}"), ",")
			{{ else -}}
				{{ .Name }} := {{ template "getParam" .Location }}("{{ .RawName }}")
			{{ end }}
		{{ end }}

		{{- if .Body -}}
			var {{ .Body.Name }} model.{{ .Body.Type }}
			if err := json.NewDecoder(r.Body).Decode(&{{ .Body.Name }}); err != nil {
				log.WithFields(log.Fields{
					"bodyType": "{{ .Body.Type }}",
					"error": err.Error(),
				}).Error("Failed to parse body data")
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := {{ .Body.Name }}.Validate(); err != nil {
				log.WithFields(log.Fields{
					"bodyType": "{{ .Body.Type }}",
					"error": err.Error(),
				}).Error("Invalid body data")
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		{{ end }}

		{{ if .ResultType }}result, {{ end }}err := m.handler.{{ .HandlerName }}({{ range .Params }}{{ .Name }}, {{ end }}{{ if .Body }}{{ .Body.Name }}{{ end }})
		if err != nil {
			message, code := m.errorTransformer.Transform(err)
			http.Error(w, message, code)
			return
		}

		{{ if .ResultType -}}
			if err := result.Validate(); err != nil {
				log.WithFields(log.Fields{
					"dataType": "{{ if .ReadOnlyResult }}ReadOnly{{ end }}{{ .ResultType }}",
					"error": err.Error(),
				}).Error("Invalid response data")
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
		{{ end -}}

		{{ if .ResultType -}}
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
