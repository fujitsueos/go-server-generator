package templates

var Swagger = parse("swagger", `
package generate

const Swagger = '
{{ . -}}
'
`)
