package generate

import (
	"errors"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/go-openapi/spec"
	log "github.com/sirupsen/logrus"
)

type routerData struct {
	Routes       []routeData
	ModelPackage string

	// meta flags, whether time and strings should be imported
	NeedsTime    bool
	NeedsStrings bool
}

type routeData struct {
	Method         string
	Route          string
	Name           string
	HandlerName    string
	Body           *bodyData
	PathParams     []paramData
	QueryParams    []paramData
	ResultType     string
	ReadOnlyResult bool
	Tag            string

	// meta-properties for the template
	NewBlock bool
}

type bodyData struct {
	Name string
	Type string
}

type paramData struct {
	Name     string
	RawName  string
	Type     string
	Required bool
	IsArray  bool
}

var routerTemplate *template.Template

func init() {
	var err error
	if routerTemplate, err = readTemplateFromFile("router", "router.tpl"); err != nil {
		logger.Fatal(err)
	}
}

// Router generates the model based on a definitions spec
func Router(w io.Writer, paths *spec.Paths, readOnlyTypes map[string]bool, modelPackage string) (err error) {
	var router routerData
	if router, err = createRouter(paths, readOnlyTypes); err != nil {
		return
	}

	router.ModelPackage = modelPackage

	err = routerTemplate.Execute(w, router)
	return
}

func createRouter(paths *spec.Paths, readOnlyTypes map[string]bool) (router routerData, err error) {
	var r routeData

	for path, pathItem := range paths.Paths {
		logger = logger.WithField("path", path)

		if pathItem.Get != nil {
			if r, err = createRouteData(http.MethodGet, path, pathItem.Get, pathItem.Parameters, readOnlyTypes); err != nil {
				return
			}

			router.Routes = append(router.Routes, r)
		}

		if pathItem.Post != nil {
			if r, err = createRouteData(http.MethodPost, path, pathItem.Post, pathItem.Parameters, readOnlyTypes); err != nil {
				return
			}

			router.Routes = append(router.Routes, r)
		}

		if pathItem.Put != nil {
			if r, err = createRouteData(http.MethodPut, path, pathItem.Put, pathItem.Parameters, readOnlyTypes); err != nil {
				return
			}

			router.Routes = append(router.Routes, r)
		}

		if pathItem.Delete != nil {
			if r, err = createRouteData(http.MethodDelete, path, pathItem.Delete, pathItem.Parameters, readOnlyTypes); err != nil {
				return
			}

			router.Routes = append(router.Routes, r)
		}

		if pathItem.Head != nil || pathItem.Options != nil || pathItem.Patch != nil {
			err = errors.New("Unsupported opteration (HEAD/OPTIONS/PATCH)")
			logger.Error(err)
			return
		}
	}

	sortRouter(router)

	prevTag := ""
	for i := range router.Routes {
		route := &router.Routes[i]

		if route.Tag != prevTag {
			route.NewBlock = true
		}

		prevTag = route.Tag

		for _, p := range append(route.PathParams, route.QueryParams...) {
			if strings.Contains(p.Type, "time.Time") {
				router.NeedsTime = true
			}
			if p.IsArray {
				router.NeedsStrings = true
			}
		}
	}

	return
}

func createRouteData(method, path string, operation *spec.Operation, routeParameters []spec.Parameter, readOnlyTypes map[string]bool) (r routeData, err error) {
	defer restoreLogger(logger)
	logger = logger.WithField("method", method)

	if operation.ID == "" {
		err = errors.New("Missing operation ID")
		logger.Error(err)
		return
	}

	if len(operation.Tags) > 1 {
		err = errors.New("Multiple tags are not supported")
		logger.Error(err)
		return
	}
	tag := "Other"
	if len(operation.Tags) == 1 {
		tag = operation.Tags[0]
	}

	handlerName := goFormat(operation.ID)
	name := lowerStart(handlerName)

	paramMap := mergeParams(routeParameters, operation.Parameters)

	if len(paramMap["header"]) > 0 || len(paramMap["formData"]) > 0 {
		err = errors.New("header and formData parameters are not supported")
		logger.Error(err)
		return
	}

	var body *bodyData
	if body, err = createBodyData(paramMap["body"]["body"]); err != nil {
		return
	}

	var pathParams, queryParams []paramData
	if pathParams, err = createparamData("path", paramMap["path"]); err != nil {
		return
	}
	if queryParams, err = createparamData("query", paramMap["query"]); err != nil {
		return
	}

	var resultType string
	var readOnlyResult bool
	if resultType, readOnlyResult, err = createResultType(operation.Responses, readOnlyTypes); err != nil {
		return
	}

	r = routeData{
		Method:         method,
		Route:          formatParams(path),
		Name:           name,
		HandlerName:    handlerName,
		Body:           body,
		PathParams:     pathParams,
		QueryParams:    queryParams,
		ResultType:     resultType,
		ReadOnlyResult: readOnlyResult,
		Tag:            tag,
	}

	return
}

func createBodyData(bodyParam *spec.Parameter) (body *bodyData, err error) {
	// no body
	if bodyParam == nil {
		return
	}

	if !bodyParam.Required {
		err = errors.New("Optional body is not supported")
		logger.Error(err)
		return
	}

	var bodyType string
	if bodyType, err = getRefName(bodyParam.Schema.Ref); err != nil {
		return
	}

	body = &bodyData{
		Name: "body" + goFormat(bodyParam.Name),
		Type: bodyType,
	}

	return
}

func createparamData(location string, params map[string]*spec.Parameter) (data []paramData, err error) {
	defer restoreLogger(logger)

	for _, param := range params {
		logger = logger.WithFields(log.Fields{
			"parameter":         param.Name,
			"parameterType":     param.Type,
			"parameterLocation": param.In,
			"parameterFormat":   param.Format,
		})

		if !(param.Type == "string" || param.Type == "array") {
			err = errors.New("Only strings, dates and arrays are supported in path and query")
			logger.Error(err)
			return
		}

		t := "string"
		isArray := false

		if param.Type == "string" {
			if !(param.Format == "" || param.Format == "date-time" || param.Format == "password") {
				err = errors.New("Unsupported string format")
				logger.Error(err)
				return
			}

			if param.Format == "date-time" {
				t = "time.Time"
			}
		} else { // "array"
			isArray = true

			if !(param.Items.Type == "string" && param.Items.Format == "") {
				err = errors.New("Only arrays of strings are supported")
				logger.Error(err)
				return
			}

			if !(param.CollectionFormat == "" || param.CollectionFormat == "csv") {
				err = errors.New("Only comma-separated arrays are supported")
				logger.Error(err)
				return
			}
		}

		data = append(data, paramData{
			Name:     location + goFormat(param.Name),
			RawName:  param.Name,
			Type:     t,
			Required: param.Required,
			IsArray:  isArray,
		})
	}

	return
}

func createResultType(responses *spec.Responses, readOnlyTypes map[string]bool) (resultType string, readOnlyResult bool, err error) {
	defer restoreLogger(logger)

	hasSuccessResponse := false
	for code, response := range responses.ResponsesProps.StatusCodeResponses {
		logger = logger.WithField("responseCode", code)

		if code >= 200 && code < 300 {
			if hasSuccessResponse {
				err = errors.New("Only one success response is supported")
				logger.Error(err)
				return
			}

			hasSuccessResponse = true

			if response.Schema != nil {
				if resultType, err = getRefName(response.Schema.Ref); err != nil {
					return
				}

				if readOnlyTypes[resultType] {
					readOnlyResult = true
				}
			}
		} else {
			if response.Schema != nil {
				err = errors.New("Non-success responses with schemas are not supported")
				logger.Error(err)
				return
			}
		}
	}

	return
}

var paramRegex = regexp.MustCompile("{([^}]*)}")

func formatParams(path string) string {
	return paramRegex.ReplaceAllString(path, ":$1")
}

// type to index parameters by type ("In") and name
type parameterMap map[string]map[string]*spec.Parameter

// Add a new parameter. Overwrites the old parameter with same type and name.
func (m *parameterMap) add(param spec.Parameter) {
	if param.In == "body" {
		// there can only be one body
		(*m)["body"] = map[string]*spec.Parameter{
			"body": &param,
		}
	} else {
		var nameMap map[string]*spec.Parameter
		var ok bool
		if nameMap, ok = (*m)[param.In]; !ok {
			nameMap = make(map[string]*spec.Parameter)
			(*m)[param.In] = nameMap
		}

		nameMap[param.Name] = &param
	}
}

// merge parameters, where later parameters overwrite earlier parameters with the same type and name
func mergeParams(params ...[]spec.Parameter) (m parameterMap) {
	m = make(parameterMap)

	for _, paramList := range params {
		for _, param := range paramList {
			m.add(param)
		}
	}

	return
}

func upperStart(s string) string {
	return strings.ToUpper(s[:1]) + s[1:]
}

func lowerStart(s string) string {
	return strings.ToLower(s[:1]) + s[1:]
}

type routeByRoute []routeData
type paramByName []paramData

var methodOrder = map[string]int{
	"GET":    0,
	"POST":   1,
	"PUT":    2,
	"DELETE": 3,
}

func (a routeByRoute) Len() int      { return len(a) }
func (a routeByRoute) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a routeByRoute) Less(i, j int) bool {
	if a[i].Tag != a[j].Tag {
		return a[i].Tag < a[j].Tag
	}

	if a[i].Route != a[j].Route {
		return a[i].Route < a[j].Route
	}

	return methodOrder[a[i].Method] < methodOrder[a[j].Method]
}

func (a paramByName) Len() int           { return len(a) }
func (a paramByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a paramByName) Less(i, j int) bool { return a[i].Name < a[j].Name }

func sortRouter(router routerData) {
	sort.Sort(routeByRoute(router.Routes))

	for _, r := range router.Routes {
		sort.Sort(paramByName(r.PathParams))
		sort.Sort(paramByName(r.QueryParams))
	}
}
