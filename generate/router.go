package generate

import (
	"errors"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/fujitsueos/go-server-generator/templates"
	"github.com/go-openapi/spec"
	log "github.com/sirupsen/logrus"
)

type routerData struct {
	Routes                       []routeData
	ModelPackage                 string
	BadRequestErrors             []string
	InternalServerErrors         []string
	AllErrors                    []errorTypeData
	HasParameterArray            bool
	HasParameterArrayValidation  bool
	HasParameterStringValidation bool
}

type routeData struct {
	Method         string
	Route          string
	Name           string
	HandlerName    string
	Body           *bodyData
	Params         []paramData
	HasPathParams  bool
	HasQueryParams bool
	HasValidation  bool

	ResultType      string
	IsResultSlice   bool
	ReadOnlyResult  bool
	ResultErrors    []errorData
	ValidationError *string
	CatchAllError   *string

	Tag string

	// meta-properties for the template
	NewBlock bool
}

type bodyData struct {
	Name string
	Type string
}

type paramData struct {
	Location       string
	Name           string
	RawName        string
	Type           string
	Validation     validation
	ItemValidation *stringValidation
	Required       bool
	IsArray        bool
}

type errorData struct {
	Type       string
	StatusCode int
}

type errorTypeData struct {
	Type        string
	PrivateType string
	Routes      []errorRouteData
}

type errorRouteData struct {
	Route        string
	PrivateRoute string
	StatusCode   int
}

// Router generates the model based on a definitions spec
func Router(routerWriter, routeErrorsWriter io.Writer, paths *spec.Paths, readOnlyTypes map[string]bool, modelPackage string) (err error) {
	var router routerData
	if router, err = createRouter(paths, readOnlyTypes); err != nil {
		return
	}

	router.ModelPackage = modelPackage

	if err = templates.Router.Execute(routerWriter, router); err != nil {
		return
	}
	err = templates.RouteErrors.Execute(routeErrorsWriter, router)
	return
}

func createRouter(paths *spec.Paths, readOnlyTypes map[string]bool) (router routerData, err error) {
	if router, err = createRouterFromPaths(paths, readOnlyTypes); err != nil {
		return
	}

	if router.BadRequestErrors, router.InternalServerErrors, err = getErrorTypes(router.Routes); err != nil {
		return
	}

	router.HasParameterArray, router.HasParameterArrayValidation, router.HasParameterStringValidation = getParametersChecks(router.Routes)

	groupErrors(&router)

	sortRouter(&router)

	prevTag := ""
	for i := range router.Routes {
		route := &router.Routes[i]

		if route.Tag != prevTag {
			route.NewBlock = true
		}

		prevTag = route.Tag
	}

	return
}

func createRouterFromPaths(paths *spec.Paths, readOnlyTypes map[string]bool) (router routerData, err error) {
	defer restoreLogger(logger)

	var r routeData

	for path, pathItem := range paths.Paths {
		logger = logger.WithField("path", path)

		operations := map[string]*spec.Operation{
			http.MethodGet:    pathItem.Get,
			http.MethodPost:   pathItem.Post,
			http.MethodPut:    pathItem.Put,
			http.MethodDelete: pathItem.Delete,
		}

		for method, operation := range operations {
			if operation != nil {
				if r, err = createRouteData(method, path, operation, pathItem.Parameters, readOnlyTypes); err != nil {
					return
				}

				router.Routes = append(router.Routes, r)
			}
		}

		if pathItem.Head != nil || pathItem.Options != nil || pathItem.Patch != nil {
			err = errors.New("Unsupported operation (HEAD/OPTIONS/PATCH)")
			logger.Error(err)
			return
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

	paramMap := mergeParams(routeParameters, operation.Parameters)
	if len(paramMap["formData"]) > 0 {
		err = errors.New("formData parameters are not supported")
		logger.Error(err)
		return
	}

	handlerName := goFormat(operation.ID)

	r = routeData{
		Method:      method,
		Route:       formatParams(path),
		Name:        lowerStart(handlerName),
		HandlerName: handlerName,
		Tag:         "Other",
	}

	if len(operation.Tags) == 1 {
		r.Tag = operation.Tags[0]
	}

	if r.Body, err = createBodyData(paramMap["body"]["body"]); err != nil {
		return
	}
	r.HasValidation = r.Body != nil

	for _, p := range []string{"path", "query", "header"} {
		var (
			params        []paramData
			hasValidation bool
		)
		if params, hasValidation, err = createParamData(p, paramMap[p]); err != nil {
			return
		}

		r.Params = append(r.Params, params...)
		if p == "path" && len(params) > 0 {
			r.HasPathParams = true
		}
		if p == "query" && len(params) > 0 {
			r.HasQueryParams = true
		}

		r.HasValidation = (r.HasValidation || hasValidation)
	}

	r.ResultType, r.IsResultSlice, r.ReadOnlyResult, r.ResultErrors, err = createResultType(operation.Responses, readOnlyTypes)
	r.ValidationError = getError(r.ResultErrors, http.StatusBadRequest)
	r.CatchAllError = getError(r.ResultErrors, http.StatusInternalServerError)

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

func createParamData(location string, params map[string]*spec.Parameter) (data []paramData, hasValidation bool, err error) {
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

		pData := paramData{
			Location: location,
			Name:     location + goFormat(param.Name),
			RawName:  param.Name,
			Required: param.Required,
			Type:     "string",
		}

		if param.Type == "string" {
			if !(param.Format == "" || param.Format == "date-time" || param.Format == "password") {
				err = errors.New("Unsupported string format")
				logger.Error(err)
				return
			}

			if param.Format == "date-time" {
				pData.Type = "time.Time"
				hasValidation = true

				if err = checkUnsupportedParamValidation(param.CommonValidations, []string{}); err != nil {
					return
				}
			} else {
				if pData.Validation, err = getParamValidation("string", param.CommonValidations); err != nil {
					return
				}

				if err = checkUnsupportedParamValidation(param.CommonValidations, []string{"minLength", "maxLength", "enum"}); err != nil {
					return
				}

				hasValidation = hasValidation || pData.Validation.String != nil
			}
		} else { // "array"
			pData.IsArray = true

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

			if pData.Validation, err = getParamValidation("array", param.CommonValidations); err != nil {
				return
			}
			var itemValidation validation
			if itemValidation, err = getParamValidation("string", param.Items.CommonValidations); err != nil {
				return
			}
			if itemValidation.String != nil {
				pData.ItemValidation = itemValidation.String
			}

			if err = checkUnsupportedParamValidation(param.CommonValidations, []string{"minItems", "maxItems", "uniqueItems"}); err != nil {
				return
			}
			if err = checkUnsupportedParamValidation(param.Items.CommonValidations, []string{"minLength", "maxLength", "enum"}); err != nil {
				return
			}

			hasValidation = hasValidation || pData.Validation.String != nil || pData.ItemValidation != nil
		}

		data = append(data, pData)
	}

	return
}

func createResultType(responses *spec.Responses, readOnlyTypes map[string]bool) (resultType string, isResultSlice bool, readOnlyResult bool, resultErrors []errorData, err error) {
	defer restoreLogger(logger)

	hasSuccessResponse := false
	resultErrorTypes := map[string]struct{}{}

	if responses.ResponsesProps.Default != nil {
		err = errors.New("Default response is not supported, use explicit response codes")
		logger.Error(err)
		return
	}

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
				schema := response.Schema
				if len(response.Schema.Type) == 1 && response.Schema.Type[0] == "array" {
					isResultSlice = true
					schema = schema.Items.Schema
				}
				if resultType, err = getRefName(schema.Ref); err != nil {
					return
				}

				if readOnlyTypes[resultType] {
					readOnlyResult = true
				}
			}
		} else {
			resultError := errorData{
				StatusCode: code,
			}

			if response.Schema != nil {
				if resultError.Type, err = getRefName(response.Schema.Ref); err != nil {
					return
				}

				if _, exists := resultErrorTypes[resultError.Type]; exists {
					// we do a switch on error type to determine the status code, so each type must be unique
					err = errors.New("Cannot have multiple status codes with the same response type")
					logger.Error(err)
					return
				}
			} else {
				resultError.Type = "string"
			}

			resultErrorTypes[resultError.Type] = struct{}{}

			resultErrors = append(resultErrors, resultError)
		}
	}

	return
}

func getParamValidation(t string, validations spec.CommonValidations) (val validation, err error) {
	switch t {
	case "string":
		stringVal := &stringValidation{}

		if validations.Enum != nil {
			stringVal.Enum = make([]string, len(validations.Enum))
			for i := range validations.Enum {
				var ok bool
				if stringVal.Enum[i], ok = validations.Enum[i].(string); !ok {
					err = errors.New("Invalid enum value")
					logger.WithField("enum", validations.Enum[i]).Error(err)
					return
				}
			}
			stringVal.FlattenedEnum = flattenEnum(stringVal.Enum, ", ", "\"")
			val.String = stringVal
		}
		if validations.MinLength != nil {
			stringVal.HasMinLength = true
			stringVal.MinLength = *validations.MinLength
			val.String = stringVal
		}
		if validations.MaxLength != nil {
			stringVal.HasMaxLength = true
			stringVal.MaxLength = *validations.MaxLength
			val.String = stringVal
		}
	case "array":
		arrayVal := &arrayValidation{}

		if validations.MinItems != nil {
			arrayVal.HasMinItems = true
			arrayVal.MinItems = *validations.MinItems
			val.Array = arrayVal
		}
		if validations.MaxItems != nil {
			arrayVal.HasMaxItems = true
			arrayVal.MaxItems = *validations.MaxItems
			val.Array = arrayVal
		}
		if validations.UniqueItems {
			arrayVal.UniqueItems = true
			val.Array = arrayVal
		}
	default:
		err = errors.New("Unsupported type")
		logger.WithField("type", t).Error(err)
	}

	return
}

func checkUnsupportedParamValidation(validations spec.CommonValidations, allowedValidations []string) (err error) {
	allowValdationsMap := make(map[string]struct{})
	for _, validation := range allowedValidations {
		allowValdationsMap[validation] = struct{}{}
	}

	unsupportedValidations := make([]string, 0)

	addUnsupportedValidation := func(validationPresent bool, name string) {
		if validationPresent {
			if _, validationAllowed := allowValdationsMap[name]; !validationAllowed {
				unsupportedValidations = append(unsupportedValidations, name)
			}
		}
	}

	// pointers
	addUnsupportedValidation(validations.Maximum != nil, "maximum")
	addUnsupportedValidation(validations.MaxItems != nil, "maxItems")
	addUnsupportedValidation(validations.MaxLength != nil, "maxLength")
	addUnsupportedValidation(validations.Minimum != nil, "minimum")
	addUnsupportedValidation(validations.MinItems != nil, "minItems")
	addUnsupportedValidation(validations.MinLength != nil, "minLength")
	addUnsupportedValidation(validations.MultipleOf != nil, "multipleOf")

	// slice / string
	addUnsupportedValidation(len(validations.Enum) > 0, "enum")
	addUnsupportedValidation(len(validations.Pattern) > 0, "pattern")

	// booleans
	addUnsupportedValidation(validations.ExclusiveMaximum, "exclusiveMaximum")
	addUnsupportedValidation(validations.ExclusiveMinimum, "exclusiveMinimum")
	addUnsupportedValidation(validations.UniqueItems, "uniqueItems")

	if len(unsupportedValidations) > 0 {
		err = errors.New("Unsupported validations")
		logger.WithField("unsupportedValidations", unsupportedValidations).Error(err)
	}

	return
}

// collect all the error types that can be returned with status code 400 and 500 from all routes combined
func getErrorTypes(routes []routeData) (validationErrors, catchAllErrors []string, err error) {
	defer restoreLogger(logger)

	validationErrorsSet := make(map[string]struct{})
	catchAllErrorsSet := make(map[string]struct{})

	hasRoutesWithValidation := false

	for _, route := range routes {
		validationError := getError(route.ResultErrors, http.StatusBadRequest)
		catchAllError := getError(route.ResultErrors, http.StatusInternalServerError)

		if route.HasValidation {
			hasRoutesWithValidation = true
			if validationError != nil {
				validationErrorsSet[*validationError] = struct{}{}
			}
		}

		if catchAllError != nil {
			catchAllErrorsSet[*catchAllError] = struct{}{}
		}
	}

	if len(validationErrorsSet) == 0 {
		// if we have routes that need validation but none of them specifies the error type, it defaults to string
		if hasRoutesWithValidation {
			validationErrors = []string{"string"}
		}
	} else {
		// if at least one route specifies the validation error type, all routes that have validation must
		for _, route := range routes {
			validationError := getError(route.ResultErrors, http.StatusBadRequest)

			if route.HasValidation && validationError == nil {
				err = errors.New("Not all routes that can have validation errors define an error type")
				logger.WithFields(log.Fields{
					"method": route.Method,
					"route":  route.Route,
				}).Error(err)
				return
			}
		}

		validationErrors = stringSetToList(validationErrorsSet)
	}

	if len(catchAllErrorsSet) == 0 {
		// if none of the routes specifies the catch-all error type it defaults to string
		catchAllErrors = []string{"string"}
	} else {
		// if at least one route specifies the catch-all error type, all of them must
		for _, route := range routes {
			if getError(route.ResultErrors, http.StatusInternalServerError) == nil {
				err = errors.New("Not all routes define a standard error type")
				logger.WithFields(log.Fields{
					"method": route.Method,
					"route":  route.Route,
				}).Error(err)
				return
			}
		}

		catchAllErrors = stringSetToList(catchAllErrorsSet)
	}

	return
}

// check if there is a parameter for some route that needs to parse/validate strings and/or arrays
func getParametersChecks(routes []routeData) (hasArray, hasArrayValidation, hasStringValidation bool) {
	for _, route := range routes {
		for _, parameter := range route.Params {
			if parameter.IsArray {
				hasArray = true

				if parameter.Validation.Array != nil {
					hasArrayValidation = true
				}

				if parameter.ItemValidation != nil {
					hasStringValidation = true
				}
			} else if parameter.Validation.String != nil {
				hasStringValidation = true
			}

			// values can never become false again; return immediately
			if hasArray && hasArrayValidation && hasStringValidation {
				return
			}
		}
	}

	return
}

func getError(errors []errorData, statusCode int) *string {
	for _, e := range errors {
		if e.StatusCode == statusCode {
			return &e.Type
		}
	}
	return nil
}

func stringSetToList(set map[string]struct{}) []string {
	result := make([]string, len(set), len(set))
	i := 0
	for s := range set {
		result[i] = s
		i++
	}
	return result
}

// match text between {}, which is the swagger notation for path parameters
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

func groupErrors(router *routerData) {
	routesMap := map[string][]errorRouteData{}

	for _, route := range router.Routes {
		for _, err := range route.ResultErrors {
			routes, ok := routesMap[err.Type]
			if !ok {
				routes = []errorRouteData{}
			}

			routesMap[err.Type] = append(routes, errorRouteData{
				Route:        route.HandlerName,
				PrivateRoute: route.Name,
				StatusCode:   err.StatusCode,
			})
		}
	}

	for name, routes := range routesMap {
		router.AllErrors = append(router.AllErrors, errorTypeData{
			Type:        name,
			PrivateType: lowerStart(name),
			Routes:      routes,
		})
	}
}

type routeByRoute []routeData
type paramByLocationAndName []paramData
type errorDataByStatusCode []errorData
type errorByType []errorTypeData
type errorByRoute []errorRouteData

var methodOrder = map[string]int{
	"GET":    0,
	"POST":   1,
	"PUT":    2,
	"DELETE": 3,
}

var locationOrder = map[string]int{
	"path":   0,
	"header": 1,
	"query":  2,
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

func (a paramByLocationAndName) Len() int      { return len(a) }
func (a paramByLocationAndName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a paramByLocationAndName) Less(i, j int) bool {
	if locationOrder[a[i].Location] != locationOrder[a[j].Location] {
		return locationOrder[a[i].Location] < locationOrder[a[j].Location]
	}

	return a[i].Name < a[j].Name
}

func (a errorDataByStatusCode) Len() int      { return len(a) }
func (a errorDataByStatusCode) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a errorDataByStatusCode) Less(i, j int) bool {
	return a[i].StatusCode < a[j].StatusCode
}

func (a errorByType) Len() int      { return len(a) }
func (a errorByType) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a errorByType) Less(i, j int) bool {
	return a[i].Type < a[j].Type
}

func (a errorByRoute) Len() int      { return len(a) }
func (a errorByRoute) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a errorByRoute) Less(i, j int) bool {
	return a[i].Route < a[j].Route
}

func sortRouter(router *routerData) {
	sort.Sort(routeByRoute(router.Routes))

	for _, r := range router.Routes {
		sort.Sort(paramByLocationAndName(r.Params))
		sort.Sort(errorDataByStatusCode(r.ResultErrors))
	}

	sort.Strings(router.BadRequestErrors)
	sort.Strings(router.InternalServerErrors)

	for i := range router.AllErrors {
		sort.Sort(errorByRoute(router.AllErrors[i].Routes))
	}
	sort.Sort(errorByType(router.AllErrors))
}
