package generate

import (
	"errors"
	"io"
	"sort"
	"text/template"

	"strings"

	"github.com/go-openapi/spec"
	log "github.com/sirupsen/logrus"
)

type modelData struct {
	Types []typeData

	// meta flag, whether time should be imported
	NeedsTime bool
}

type typeData struct {
	Name        string
	Description string
	Type        string
	Props       []propsData
	RefType     string
}

type propsData struct {
	Name        string
	Type        string
	JSONName    string
	Description string
}

var modelTemplate *template.Template

func init() {
	var err error
	if modelTemplate, err = readTemplateFromFile("model", "model.tpl"); err != nil {
		log.Fatal(err)
	}
}

// Model generates the model based on a definitions spec
func Model(w io.Writer, definitions spec.Definitions) (err error) {
	var model modelData
	if model, err = createModel(definitions); err != nil {
		return
	}

	err = modelTemplate.Execute(w, model)
	return
}

func createModel(definitions spec.Definitions) (model modelData, err error) {
	for name, definition := range definitions {
		logger = log.WithFields(log.Fields{
			"definition": name,
		})

		if len(definition.Type) != 1 || (definition.Type[0] != "object" && definition.Type[0] != "array") {
			err = errors.New("Unexpected definition type")
			logger.WithField("type", definition.Type).Error(err)
			return
		}

		logger = logger.WithField("type", definition.Type[0])

		t := typeData{
			Name:        goFormat(name),
			Description: definition.Description,
		}

		if definition.Type[0] == "object" {
			t.Type = "struct"
			if t.Props, err = createObjectProps(definition); err != nil {
				return
			}
		} else { // "array"
			t.Type = "slice"
			if t.RefType, err = getRefName(definition.Items.Schema.Ref); err != nil {
				return
			}
		}

		model.Types = append(model.Types, t)
	}

	sortModel(model)

	for _, t := range model.Types {
		for _, p := range t.Props {
			if strings.Contains(p.Type, "time.Time") {
				model.NeedsTime = true
				return
			}
		}
	}

	return
}

func createObjectProps(definition spec.Schema) (props []propsData, err error) {
	defer restoreLogger(logger)

	for propName, property := range definition.Properties {
		logger = logger.WithFields(log.Fields{
			"property":     propName,
			"propertyType": property.Type,
		})

		if len(property.Type) > 1 {
			err = errors.New("Unexpected property type")
			logger.Error(err)
			return
		}

		var propType string
		if propType, err = getType(property); err != nil {
			return
		}

		props = append(props, propsData{
			Name:        goFormat(propName),
			JSONName:    propName,
			Type:        propType,
			Description: property.Description,
		})
	}

	return
}

var primitiveTypes = map[string]string{
	"boolean": "bool",
	"integer": "int",
	"number":  "double",
	"string":  "string",
}

func getType(schema spec.Schema) (t string, err error) {
	propertyType := "ref"
	if len(schema.Type) == 1 {
		propertyType = schema.Type[0]
	}

	var ok bool
	if t, ok = primitiveTypes[propertyType]; ok {
		if t == "string" && schema.Format != "" {
			if schema.Format != "date-time" && schema.Format != "password" {
				err = errors.New("Unsupported string format")
				logger.Error(err)
				return
			}

			if schema.Format == "date-time" {
				t = "time.Time"
			}
		}

		return
	}

	if propertyType == "array" {
		if schema.Items == nil {
			err = errors.New("Array does not have items")
			return
		}

		if schema.Items.Schema == nil {
			err = errors.New("Array does not have a single type")
			return
		}

		var subType string
		if subType, err = getType(*schema.Items.Schema); err != nil {
			return
		}

		t = "[]" + subType
		return
	}

	if propertyType == "ref" {
		defer restoreLogger(logger)
		logger = logger.WithField("schema", schema.ID)

		t, err = getRefName(schema.Ref)
		return
	}

	err = errors.New("Unsupported type")
	logger.Error(err)
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
