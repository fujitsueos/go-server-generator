package templates

// RouteErrors is a template for the error types of each route
var RouteErrors = parse("routeErrors", `
package model

// This is a generated file
// Manual changes will be overwritten

{{ range .Routes -}}
	{{ $route := . -}}
	// {{ .HandlerName }}Error is implemented by:
	{{ range .ResultErrors -}}
		// | {{ .StatusCode }}: {{ .Type }}
	{{ end -}}
	type {{ .HandlerName }}Error interface {
		{{ .Name }}Error() {{ .Name }}Error
		{{ .HandlerName }}StatusCode() (t string, statusCode int)
	}

	type {{ .Name }}Error byte

	{{ range .ResultErrors -}}
		func (e *{{ .Type }}) {{ $route.Name }}Error() {{ $route.Name }}Error {
			return {{ $route.Name }}Error(0)
		}

		func (e *{{ .Type }}) {{ $route.HandlerName }}StatusCode() (t string, statusCode int) {
			return "{{ .Type }}", {{ .StatusCode }}
		}
	{{ end -}}
{{ end -}}
`)
