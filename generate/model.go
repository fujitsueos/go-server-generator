package generate

import (
	"errors"
	"io"
	"sort"
	"strings"
	"text/template"

	"github.com/go-openapi/spec"
	log "github.com/sirupsen/logrus"
)

type modelData struct {
	Types []typeData

	// meta flag, whether time should be imported
	NeedsTime bool
}

type typeData struct {
	// general fields
	Name             string
	Description      string
	Validation       validation
	HasReadOnlyProps bool

	// struct fields
	IsStruct bool
	Props    []propsData

	// slice fields
	IsSlice        bool
	ItemType       string
	ItemValidation validation

	// primitive type fields
	Type string
}

type propsData struct {
	Name        string
	JSONName    string
	Description string
	Validation  validation
	IsReadOnly  bool

	// actually a validation field, but this is easier for the template
	IsRequired bool

	// slice fields
	IsSlice        bool
	ItemType       string
	ItemValidation validation

	// primitive type and reference fields
	Type string
}

var modelTemplate *template.Template

func init() {
	var err error
	if modelTemplate, err = readTemplateFromFile("model", "model.tpl"); err != nil {
		logger.Fatal(err)
	}
}

// Model generates the model based on a definitions spec
func Model(modelWriter io.Writer, validateWriter io.Writer, definitions spec.Definitions) (readOnlyTypes map[string]bool, err error) {
	var model modelData
	if model, readOnlyTypes, err = createModel(definitions); err != nil {
		return
	}

	if err = modelTemplate.Execute(modelWriter, model); err != nil {
		return
	}

	err = validateTemplate.Execute(validateWriter, model)

	return
}

func createModel(definitions spec.Definitions) (model modelData, readOnlyTypes map[string]bool, err error) {
	originalLogger := logger

	for name, definition := range definitions {
		logger = originalLogger.WithFields(log.Fields{
			"definition": name,
		})

		logger.Info("Generating model")

		var (
			goType       string
			val, itemVal validation
			isSlice      bool
		)

		if goType, val, itemVal, isSlice, err = getType(definition); err != nil {
			return
		}

		logger = logger.WithField("type", goType)

		t := typeData{
			Name:        goFormat(name),
			Description: definition.Description,
			Validation:  val,
		}

		if goType == "struct" {
			t.IsStruct = true

			required := []string{}
			if val.Object != nil {
				required = val.Object.Required
			}

			if t.Props, t.HasReadOnlyProps, err = createObjectProps(definition, required); err != nil {
				return
			}
		} else if isSlice {
			t.IsSlice = true
			t.ItemType = goType
			t.ItemValidation = itemVal
		} else {
			t.Type = goType
		}

		model.Types = append(model.Types, t)
	}

	logger = originalLogger

	if readOnlyTypes, err = checkReadOnlyTypes(&model); err != nil {
		return
	}

	sortModel(model)

	for _, t := range model.Types {
		if t.Type == "time.Time" {
			model.NeedsTime = true
			return
		}

		for _, p := range t.Props {
			if strings.Contains(p.Type, "time.Time") {
				model.NeedsTime = true
				return
			}
		}
	}

	return
}

func createObjectProps(definition spec.Schema, requiredProps []string) (props []propsData, hasReadOnlyProps bool, err error) {
	defer restoreLogger(logger)

	requiredMap := map[string]bool{}
	for _, requiredProp := range requiredProps {
		requiredMap[requiredProp] = true
	}

	originalLogger := logger

	for propName, property := range definition.Properties {
		logger = originalLogger.WithFields(log.Fields{
			"property":     propName,
			"propertyType": property.Type,
		})

		logger.Info("Generating property")

		var (
			goType       string
			isSlice      bool
			val, itemVal validation
		)

		if goType, val, itemVal, isSlice, err = getType(property); err != nil {
			return
		}

		isRequired := requiredMap[propName]
		if !isRequired && val.hasValidation() {
			// no validation can pass if the property value is not present
			// enforce this here to make the template a bit simpler
			err = errors.New("Properties with validation must be required")
			logger.Error(err)
			return
		}

		p := propsData{
			Name:        goFormat(propName),
			JSONName:    propName,
			Description: property.Description,
			Validation:  val,
			IsReadOnly:  property.ReadOnly,
			IsRequired:  isRequired,
		}

		hasReadOnlyProps = hasReadOnlyProps || p.IsReadOnly

		if goType == "struct" {
			err = errors.New("Nested objects are not supported; use references instead")
			logger.Error(err)
			return
		} else if isSlice {
			p.IsSlice = true
			p.ItemType = goType
			p.ItemValidation = itemVal
		} else {
			p.Type = goType
		}

		props = append(props, p)
	}

	return
}

var primitiveTypes = map[string]string{
	"boolean": "bool",
	"integer": "int64",
	"number":  "float64",
	"string":  "string",
}

func getType(schema spec.Schema) (t string, val, itemVal validation, isSlice bool, err error) {
	defer restoreLogger(logger)

	if len(schema.Type) > 1 {
		err = errors.New("Union types are not supported")
		logger.WithField("schemaType", strings.Join(schema.Type, ", ")).Error(err)
		return
	}

	if len(schema.Type) == 0 {
		logger = logger.WithField("schema", schema.ID)

		// a schema without type must have a reference
		t, err = getRefName(schema.Ref)
		return
	}

	schemaType := schema.Type[0]
	logger = logger.WithField("schemaType", schemaType)

	var ok bool
	if t, ok = primitiveTypes[schemaType]; ok {
		if t == "string" && schema.Format != "" {
			if schema.Format != "date-time" && schema.Format != "password" {
				err = errors.New("Unsupported string format")
				logger.WithField("format", schema.Format).Error(err)
				return
			}

			if schema.Format == "date-time" {
				t = "time.Time"
			}
		}
	} else if schemaType == "object" {
		t = "struct"
	} else if schemaType == "array" {
		isSlice = true

		if schema.Items == nil {
			err = errors.New("Array does not have items")
			return
		}

		if schema.Items.Schema == nil {
			err = errors.New("Array does not have a single type")
			return
		}

		var itemIsSlice bool
		if t, itemVal, _, itemIsSlice, err = getType(*schema.Items.Schema); err != nil {
			return
		}
		if itemIsSlice {
			err = errors.New("Nested arrays are not supported; use references instead")
			logger.Error(err)
			return
		}
	} else {
		err = errors.New("Unknown schema type")
		logger.Error(err)
		return
	}

	val, err = getValidationForType(t, isSlice, schema)

	return
}

// Swagger objects with read-only properties lead to two Go structs, one with the read-only
// properties and one with the rest. If we reference such an object from another object, we need
// to also create two Go structs for that one. Not impossible, but it leads to annoying bookkeeping.
//
// For now we just forbid that case: an object with read-only properties cannot be refenced by
// another object. There is one exception: top-level arrays may reference objects with read-only
// properties. This is the typical response for the GET /<resources> endpoint, so forbidding that
// makes no sense.
func checkReadOnlyTypes(model *modelData) (readOnlyTypes map[string]bool, err error) {
	defer restoreLogger(logger)

	// names of types that have read-only properties
	readOnlyTypes = make(map[string]bool)
	for _, t := range model.Types {
		if t.HasReadOnlyProps {
			readOnlyTypes[t.Name] = true
		}
	}

	readOnlyArrays := make(map[string]bool)

	// check that read-only objects are only referenced by top-level arrays
	for i := range model.Types {
		t := &model.Types[i]

		logger = logger.WithFields(log.Fields{
			"type": t.Name,
		})

		for _, dep := range getDependencies(t) {
			logger = logger.WithFields(log.Fields{
				"reference": dep,
			})

			if readOnlyTypes[dep] {
				if t.IsSlice {
					// the only allowed case: a top level array referencing a type with read-only properties
					t.HasReadOnlyProps = true
					readOnlyArrays[t.Name] = true
					readOnlyTypes[t.Name] = true
				} else {
					err = errors.New("Cannot $ref a type with read-only properties")
					logger.Error(err)
					return
				}
			}
		}
	}

	// verify that the top-level arrays are not referenced again
	for _, t := range model.Types {
		for _, dep := range getDependencies(&t) {
			if readOnlyArrays[dep] {
				err = errors.New("Cannot $ref a read-only array")
				logger.Error(err)
				return
			}
		}
	}

	return
}

func getDependencies(t *typeData) (dependencies []string) {
	switch {
	case t.IsStruct:
		dependencies = make([]string, len(t.Props))

		for i, p := range t.Props {
			if p.IsSlice {
				dependencies[i] = p.ItemType
			} else {
				dependencies[i] = p.Type
			}
		}
	case t.IsSlice:
		dependencies = []string{t.ItemType}
	default:
		dependencies = []string{t.Type}
	}

	return
}

type typeByName []typeData
type propByName []propsData

func (a typeByName) Len() int           { return len(a) }
func (a typeByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a typeByName) Less(i, j int) bool { return a[i].Name < a[j].Name }

func (a propByName) Len() int           { return len(a) }
func (a propByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a propByName) Less(i, j int) bool { return a[i].Name < a[j].Name }

func sortModel(model modelData) {
	sort.Sort(typeByName(model.Types))

	for _, t := range model.Types {
		sort.Sort(propByName(t.Props))
	}
}
