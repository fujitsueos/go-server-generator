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

	func (e *{{ .PrivateType }}Impl) {{ .PrivateType }}(){}

	const {{ .PrivateType }}Message = "{{ .Type }}"

	{{ $privateType := .PrivateType }}
	{{ range .Routes -}}
		func (e *{{ $privateType }}Impl) {{ .PrivateRoute }} (){}
		func (e *{{ $privateType }}Impl) {{ .Route }}StatusCode() (t string, statusCode int) {
			return {{ $privateType }}Message, {{ .StatusCode }}
		}
	{{ end -}}
{{ end -}}
`)
