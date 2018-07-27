package templates

var Swagger = parse("swagger", `
package generated

const Swagger = '
{{ . -}}
'
`)
