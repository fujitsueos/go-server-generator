package router

// This is a generated file
// Manual changes will be overwritten

import (
	"encoding/json"
	"net/http"
{{- if .NeedsStrings }}
	"strings"
{{- end }}
{{- if .NeedsTime }}
	"time"
{{- end }}

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"

	"{{ .ModelPackage }}"
)

// Handler implements the actual functionality of the service
type Handler interface {
{{ range .Routes -}}
{{ if and (not .First) .NewBlock }}
{{ end -}}
{{ if .NewBlock }}	// {{ .Tag }}
{{ end -}}
{{ "	" }}{{ .HandlerName }}(
	{{- range .PathParams }}{{ .Name }} {{ if .IsArray }}[]{{ end }}{{ .Type }}, {{ end }}
	{{- range .QueryParams }}{{ .Name }} {{ if .IsArray }}[]{{ end }}{{ .Type }}, {{ end }}
	{{- if .Body }}{{ .Body.Name }} model.{{ .Body.Type }}{{ end -}}
) ({{ if .ResultType }}model.{{ .ResultType }}, {{ end }}error)
{{- if not .Last }}
{{ end }}
{{- end }}
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
{{ if and (not .First) .NewBlock }}
{{ end -}}
{{ "	" }}router.{{ .Method }}("{{ .Route }}", m.{{ .Name }})
{{- if not .Last }}
{{ end -}}
{{ end }}

	return router
}

{{ range .Routes -}}
func (m *middleware) {{ .Name }}(w http.ResponseWriter, r *http.Request, {{ if .PathParams }}params{{ else }}_{{ end }} httprouter.Params) {
{{ range .PathParams }}
{{- if eq .Type "time.Time" -}}
	{{ "	" }}{{ .Name }}, parseErr := time.Parse(time.RFC3339, params.ByName("{{ .RawName }}"))
	if parseErr != nil {
		http.Error(w, "Cannot parse date", http.StatusBadRequest)
		return
	}
{{- else if .IsArray -}}
	{{ "	" }}{{ .Name }} := strings.Split(params.ByName("{{ .RawName }}"), ",")
{{- else -}}
	{{ "	" }}{{ .Name }} := params.ByName("{{ .RawName }}")
{{ end }}
{{- end }}
{{- if .PathParams }}
{{ end -}}

{{ if .QueryParams }}	query := r.URL.Query()
{{ range .QueryParams -}}
{{ "	" }}{{ .Name }} := query.Get("{{ .RawName }}")
{{ end }}
{{ end -}}

{{ if .Body -}}
{{ "	" }}var {{ .Body.Name }} model.{{ .Body.Type }}
	if err := json.NewDecoder(r.Body).Decode(&{{ .Body.Name }}); err != nil {
		log.WithFields(log.Fields{
			"bodyType": "{{ .Body.Type }}",
			"error": err.Error(),
		}).Error("Failed to parse body data")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

{{ end -}}

{{ "	"	}}{{ if .ResultType }}result, {{ end }}err := m.handler.{{ .HandlerName }}({{ range .PathParams }}{{ .Name }}, {{ end }}{{ range .QueryParams }}{{ .Name }}, {{ end }}{{ if .Body }}{{ .Body.Name }}{{ end }})
	if err != nil {
		message, code := m.errorTransformer.Transform(err)
		http.Error(w, message, code)
		return
	}

{{ if .ResultType -}}
	{{ "	" }}respondJSON(w, result, "{{ .ResultType }}")
{{- else -}}
	{{ "	" }}w.Write([]byte("OK"))
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
