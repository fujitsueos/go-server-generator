package templates

// RouteErrors is a template for the error types of each route
var RouteErrors = parse("routeErrors", `
package model

// This is a generated file
// Manual changes will be overwritten

{{ range .Routes -}}
	// {{ .HandlerName }}Error is implemented by:
	{{ range .ResultErrors -}}
		// | {{ .StatusCode }}: {{ .Type }}
	{{ end -}}
	type {{ .HandlerName }}Error interface {
		{{ .Name }}()
		{{ .HandlerName }}StatusCode() (t string, statusCode int)
	}
{{ end -}}

{{ range .AllErrors -}}
	type {{ .Type }} interface {
		error
		{{ range .Routes -}}
			{{ .Route }}Error
		{{ end -}}
		{{ .PrivateType }}()
	}
{{ end -}}

{{ range .AllErrors -}}
	func (e *{{ .PrivateType }}Impl) {{ .PrivateType }}(){}

	{{ $error := . }}
	{{ range .Routes -}}
		func (e *{{ $error.PrivateType }}Impl) {{ .PrivateRoute }} (){}
		func (e *{{ $error.PrivateType }}Impl) {{ .Route }}StatusCode() (t string, statusCode int) {
			return "{{ $error.Type }}", {{ .StatusCode }}
		}
	{{ end -}}
{{ end -}}
`)
