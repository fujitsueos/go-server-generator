package generate

import (
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/go-openapi/spec"
)

// we cannot type cast in templates, so we need to provide all possible validations
// and just only use the right one in the template
// if a pointer is non-nil there is at least one validation rule set
type validation struct {
	Object *objectValidation
	Array  *arrayValidation
	Int    *intValidation
	Number *numberValidation
	String *stringValidation
}

type objectValidation struct {
	Required []string
}

type arrayValidation struct {
	HasMaxItems bool
	MaxItems    int64
	HasMinItems bool
	MinItems    int64
}

type intValidation struct {
	Enum             []int64
	FlattenedEnum    string
	HasMaximum       bool
	ExclusiveMaximum bool
	Maximum          int64
	HasMinimum       bool
	ExclusiveMinimum bool
	Minimum          int64
}

type numberValidation struct {
	Enum             []float64
	FlattenedEnum    string
	HasMaximum       bool
	ExclusiveMaximum bool
	Maximum          float64
	HasMinimum       bool
	ExclusiveMinimum bool
	Minimum          float64
}

type stringValidation struct {
	Enum          []string
	FlattenedEnum string
	HasMaxLength  bool
	MaxLength     int64
	HasMinLength  bool
	MinLength     int64
}

var validateTemplate *template.Template

func init() {
	var err error
	if validateTemplate, err = readTemplateFromFile("validate", "validate.tpl"); err != nil {
		logger.Fatal(err)
	}
}

func getValidationForType(t string, isSlice bool, schema spec.Schema) (val validation, err error) {
	if isSlice {
		arrayVal := &arrayValidation{}

		if schema.MinItems != nil {
			arrayVal.HasMinItems = true
			arrayVal.MinItems = *schema.MinItems
			val.Array = arrayVal
		}
		if schema.MaxItems != nil {
			arrayVal.HasMaxItems = true
			arrayVal.MaxItems = *schema.MaxItems
			val.Array = arrayVal
		}
		err = checkUnsupportedFields(t, schema, []string{"items", "minItems", "maxItems"})

		return
	}

	switch t {
	case "int64":
		intVal := &intValidation{}

		if schema.Enum != nil {
			intVal.Enum = make([]int64, len(schema.Enum))
			for i := range schema.Enum {
				number, ok := schema.Enum[i].(float64)
				if !ok {
					err = errors.New("Invalid enum value")
					logger.WithField("enum", schema.Enum[i]).Error(err)
					return
				}

				intVal.Enum[i] = int64(number)
				if float64(intVal.Enum[i]) != number {
					err = errors.New("Number is not an integer")
					logger.WithField("enum", schema.Enum[i]).Error(err)
					return
				}
			}
			intVal.FlattenedEnum = flattenEnum(intVal.Enum, ", ", "")
			val.Int = intVal
		}
		if schema.Maximum != nil {
			intVal.HasMaximum = true
			intVal.ExclusiveMaximum = schema.ExclusiveMaximum
			intVal.Maximum = int64(*schema.Maximum)
			val.Int = intVal
		}
		if schema.Minimum != nil {
			intVal.HasMinimum = true
			intVal.ExclusiveMinimum = schema.ExclusiveMinimum
			intVal.Minimum = int64(*schema.Minimum)
			val.Int = intVal
		}
		err = checkUnsupportedFields(t, schema, []string{"enum", "maximum", "exclusiveMaximum", "minimum", "exclusiveMinimum", "readOnly"})
	case "float64":
		numberVal := &numberValidation{}

		if schema.Enum != nil {
			numberVal.Enum = make([]float64, len(schema.Enum))
			for i := range schema.Enum {
				var ok bool
				if numberVal.Enum[i], ok = schema.Enum[i].(float64); !ok {
					err = errors.New("Invalid enum value")
					logger.WithField("enum", schema.Enum[i]).Error(err)
					return
				}
			}
			numberVal.FlattenedEnum = flattenEnum(numberVal.Enum, ", ", "")
			val.Number = numberVal
		}
		if schema.Maximum != nil {
			numberVal.HasMaximum = true
			numberVal.ExclusiveMaximum = schema.ExclusiveMaximum
			numberVal.Maximum = *schema.Maximum
			val.Number = numberVal
		}
		if schema.Minimum != nil {
			numberVal.HasMinimum = true
			numberVal.ExclusiveMinimum = schema.ExclusiveMinimum
			numberVal.Minimum = *schema.Minimum
			val.Number = numberVal
		}
		err = checkUnsupportedFields(t, schema, []string{"enum", "maximum", "exclusiveMaximum", "minimum", "exclusiveMinimum", "readOnly"})
	case "string":
		stringVal := &stringValidation{}

		if schema.Enum != nil {
			stringVal.Enum = make([]string, len(schema.Enum))
			for i := range schema.Enum {
				var ok bool
				if stringVal.Enum[i], ok = schema.Enum[i].(string); !ok {
					err = errors.New("Invalid enum value")
					logger.WithField("enum", schema.Enum[i]).Error(err)
					return
				}
			}
			stringVal.FlattenedEnum = flattenEnum(stringVal.Enum, ", ", "\"")
			val.String = stringVal
		}
		if schema.MinLength != nil {
			stringVal.HasMinLength = true
			stringVal.MinLength = *schema.MinLength
			val.String = stringVal
		}
		if schema.MaxLength != nil {
			stringVal.HasMaxLength = true
			stringVal.MaxLength = *schema.MaxLength
			val.String = stringVal
		}
		err = checkUnsupportedFields(t, schema, []string{"enum", "format", "minLength", "maxLength", "readOnly"})
	case "bool":
		err = checkUnsupportedFields(t, schema, []string{"readOnly"})
	case "time.Time":
		err = checkUnsupportedFields(t, schema, []string{"readOnly"})
	case "struct":
		if schema.Required != nil {
			val.Object = &objectValidation{
				Required: schema.Required,
			}
		}
		err = checkUnsupportedFields(t, schema, []string{"properties", "readOnly", "required"})
	default:
		err = errors.New("Unknown type")
		logger.Error(err)
	}

	return
}

func checkUnsupportedFields(schemaType string, schema spec.Schema, allowedFields []string) (err error) {
	allowedFieldsMap := make(map[string]struct{})
	for _, field := range allowedFields {
		allowedFieldsMap[field] = struct{}{}
	}

	unsupportedFields := make([]string, 0)

	addUnsupportedField := func(fieldPresent bool, name string) {
		if fieldPresent {
			if _, fieldAllowed := allowedFieldsMap[name]; !fieldAllowed {
				unsupportedFields = append(unsupportedFields, name)
			}
		}
	}

	// pointers
	addUnsupportedField(schema.AdditionalItems != nil, "additionalItems")
	addUnsupportedField(schema.AdditionalProperties != nil, "additionalProperties")
	addUnsupportedField(schema.Default != nil, "default")
	addUnsupportedField(schema.Enum != nil, "enum")
	addUnsupportedField(schema.Items != nil, "items")
	addUnsupportedField(schema.Maximum != nil, "maximum")
	addUnsupportedField(schema.MaxItems != nil, "maxItems")
	addUnsupportedField(schema.MaxLength != nil, "maxLength")
	addUnsupportedField(schema.MaxProperties != nil, "maxProperties")
	addUnsupportedField(schema.Minimum != nil, "minimum")
	addUnsupportedField(schema.MinItems != nil, "minItems")
	addUnsupportedField(schema.MinLength != nil, "minLength")
	addUnsupportedField(schema.MinProperties != nil, "minProperties")
	addUnsupportedField(schema.MultipleOf != nil, "multipleOf")
	addUnsupportedField(schema.Not != nil, "not")

	// slices / maps / strings
	addUnsupportedField(len(schema.AllOf) > 0, "allOf")
	addUnsupportedField(len(schema.AnyOf) > 0, "anyOf")
	addUnsupportedField(len(schema.Definitions) > 0, "definitions")
	addUnsupportedField(len(schema.Dependencies) > 0, "dependencies")
	addUnsupportedField(len(schema.Extensions) > 0, "extensions")
	addUnsupportedField(len(schema.ExtraProps) > 0, "extraProps")
	addUnsupportedField(len(schema.Format) > 0, "format")
	addUnsupportedField(len(schema.OneOf) > 0, "oneOf")
	addUnsupportedField(len(schema.Pattern) > 0, "pattern")
	addUnsupportedField(len(schema.PatternProperties) > 0, "patternProperties")
	addUnsupportedField(len(schema.Properties) > 0, "properties")
	addUnsupportedField(len(schema.Required) > 0, "required")

	// booleans
	addUnsupportedField(schema.ExclusiveMaximum, "exclusiveMaximum")
	addUnsupportedField(schema.ExclusiveMinimum, "exclusiveMinimum")
	addUnsupportedField(schema.ReadOnly, "readOnly")
	addUnsupportedField(schema.UniqueItems, "uniqueItems")

	if len(unsupportedFields) > 0 {
		err = errors.New("Unsupported fields")
		logger.WithField("unsupportedFields", unsupportedFields).Error(err)
	}

	return
}

func (v validation) hasValidation() bool {
	return v.Object != nil ||
		v.Array != nil ||
		v.Int != nil ||
		v.Number != nil ||
		v.String != nil
}

// roughly the same as strings.Join, but allow quoting strings
// ([1, 2, 3], ", ", "") --> "1, 2, 3"
// (["a", "b", "c"], ",", "\""") --> "\"a\", \"b\", \"c\""
func flattenEnum(enum interface{}, separator, wrapper string) string {
	items := strings.Fields(strings.Trim(fmt.Sprint(enum), "[]"))

	return wrapper + strings.Join(items, wrapper+separator+wrapper) + wrapper
}
