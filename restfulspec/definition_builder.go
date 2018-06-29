package restfulspec

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/tangblue/goapi/spec"
)

type definitionBuilder struct {
	Definitions spec.Definitions
	Config      Config
}

// Documented is
type Documented interface {
	SwaggerDoc() map[string]string
}

// Check if this structure has a method with signature func (<theModel>) SwaggerDoc() map[string]string
// If it exists, retrieve the documentation and overwrite all struct tag descriptions
func getDocFromMethodSwaggerDoc2(model reflect.Type) map[string]string {
	if docable, ok := reflect.New(model).Elem().Interface().(Documented); ok {
		return docable.SwaggerDoc()
	}
	return make(map[string]string)
}

func (b *definitionBuilder) getDefinitions() spec.Definitions {
	return b.Definitions
}

func (b *definitionBuilder) SchemaFromModel(model reflect.Type, modelName, jsonName string) *spec.Schema {
	ret := new(spec.Schema)
	s := ret
	if model.Kind() == reflect.Array || model.Kind() == reflect.Slice {
		model = model.Elem()
		s = new(spec.Schema)
		ret.Type = []string{"array"}
		ret.Items = &spec.SchemaOrArray{Schema: s}
	}
	if model.Kind() == reflect.Ptr {
		model = model.Elem()
	}

	name := model.Kind().String()
	if isPrimitiveType(name) {
		s.AddType(jsonSchemaType(name), jsonSchemaFormat(name))
	} else {
		name = model.String()
		if name == "" {
			name = modelName + "." + jsonName
		}
		s.Ref = b.createRef(model, name)
	}

	return ret
}

// addModelFrom creates and adds a Schema to the builder and detects and calls
// the post build hook for customizations
func (b *definitionBuilder) addModelFrom(sample interface{}) {
	b.addModel(reflect.TypeOf(sample), "")
}

func (b *definitionBuilder) addModel(st reflect.Type, nameOverride string) *spec.Schema {
	// Turn pointers into simpler types so further checks are
	// correct.
	if st.Kind() == reflect.Ptr {
		st = st.Elem()
	}

	modelName := b.keyFrom(st)
	if nameOverride != "" {
		modelName = nameOverride
	}
	// no models needed for primitive types
	if b.isPrimitiveType(modelName) {
		return nil
	}
	// golang encoding/json packages says array and slice values encode as
	// JSON arrays, except that []byte encodes as a base64-encoded string.
	// If we see a []byte here, treat it at as a primitive type (string)
	// and deal with it in buildArrayTypeProperty.
	if (st.Kind() == reflect.Slice || st.Kind() == reflect.Array) &&
		st.Elem().Kind() == reflect.Uint8 {
		return nil
	}
	// see if we already have visited this model
	if _, ok := b.Definitions[modelName]; ok {
		return nil
	}
	sm := spec.Schema{
		SchemaProps: spec.SchemaProps{
			Required:   []string{},
			Properties: map[string]spec.Schema{},
		},
	}

	// reference the model before further initializing (enables recursive structs)
	b.Definitions[modelName] = sm

	// check for slice or array
	if st.Kind() == reflect.Slice || st.Kind() == reflect.Array {
		st = st.Elem()
	}
	// check for structure or primitive type
	if st.Kind() != reflect.Struct {
		return &sm
	}

	fullDoc := getDocFromMethodSwaggerDoc2(st)
	modelDescriptions := []string{}

	for i := 0; i < st.NumField(); i++ {
		field := st.Field(i)
		jsonName, modelDescription, prop := b.buildProperty(field, &sm, modelName)
		if len(modelDescription) > 0 {
			modelDescriptions = append(modelDescriptions, modelDescription)
		}

		// add if not omitted
		if len(jsonName) != 0 {
			// update description
			if fieldDoc, ok := fullDoc[jsonName]; ok {
				prop.Description = fieldDoc
			}
			// update Required
			if b.isPropertyRequired(field) {
				sm.Required = append(sm.Required, jsonName)
			}
			sm.Properties[jsonName] = prop
		}
	}

	// We always overwrite documentation if SwaggerDoc method exists
	// "" is special for documenting the struct itself
	if modelDoc, ok := fullDoc[""]; ok {
		sm.Description = modelDoc
	} else if len(modelDescriptions) != 0 {
		sm.Description = strings.Join(modelDescriptions, "\n")
	}
	// Needed to pass openapi validation. This field exists for json-schema compatibility,
	// but it conflicts with the openapi specification.
	// See https://github.com/go-openapi/spec/issues/23 for more context
	sm.ID = ""

	// update model builder with completed model
	b.Definitions[modelName] = sm

	return &sm
}

func (b *definitionBuilder) isPropertyRequired(field reflect.StructField) bool {
	required := true
	if optionalTag := field.Tag.Get("optional"); optionalTag == "true" {
		return false
	}
	if jsonTag := field.Tag.Get("json"); jsonTag != "" {
		s := strings.Split(jsonTag, ",")
		if len(s) > 1 && s[1] == "omitempty" {
			return false
		}
	}
	return required
}

func (b *definitionBuilder) buildProperty(field reflect.StructField, model *spec.Schema, modelName string) (jsonName, modelDescription string, prop spec.Schema) {
	jsonName = b.jsonNameOfField(field)
	if len(jsonName) == 0 {
		// empty name signals skip property
		return "", "", prop
	}

	if field.Name == "XMLName" && field.Type.String() == "xml.Name" {
		// property is metadata for the xml.Name attribute, can be skipped
		return "", "", prop
	}

	if tag := field.Tag.Get("modelDescription"); tag != "" {
		modelDescription = tag
	}

	setPropertyMetadata(&prop, field)
	if prop.Type != nil {
		return jsonName, modelDescription, prop
	}
	fieldType := field.Type

	// check if type is doing its own marshalling
	marshalerType := reflect.TypeOf((*json.Marshaler)(nil)).Elem()
	if fieldType.Implements(marshalerType) {
		var pType = "string"
		if prop.Type == nil {
			prop.Type = []string{pType}
		}
		if prop.Format == "" {
			prop.Format = b.jsonSchemaFormat(b.keyFrom(fieldType))
		}
		return jsonName, modelDescription, prop
	}

	// check if annotation says it is a string
	if jsonTag := field.Tag.Get("json"); jsonTag != "" {
		s := strings.Split(jsonTag, ",")
		if len(s) > 1 && s[1] == "string" {
			stringt := "string"
			prop.Type = []string{stringt}
			return jsonName, modelDescription, prop
		}
	}

	fieldKind := fieldType.Kind()
	switch {
	case fieldKind == reflect.Struct:
		jsonName, prop := b.buildStructTypeProperty(field, jsonName, model)
		return jsonName, modelDescription, prop
	case fieldKind == reflect.Slice || fieldKind == reflect.Array:
		jsonName, prop := b.buildArrayTypeProperty(field, jsonName, modelName)
		return jsonName, modelDescription, prop
	case fieldKind == reflect.Ptr:
		jsonName, prop := b.buildPointerTypeProperty(field, jsonName, modelName)
		return jsonName, modelDescription, prop
	case fieldKind == reflect.String:
		stringt := "string"
		prop.Type = []string{stringt}
		return jsonName, modelDescription, prop
	case fieldKind == reflect.Map:
		jsonName, prop := b.buildMapTypeProperty(field, jsonName, modelName)
		return jsonName, modelDescription, prop
	}

	prop = *b.SchemaFromModel(fieldType, modelName, jsonName)
	setPropertyMetadata(&prop, field)
	return jsonName, modelDescription, prop
}

func (b *definitionBuilder) createRef(st reflect.Type, name string) spec.Ref {
	b.addModel(st, name)
	return spec.MustCreateRef("#/definitions/" + name)
}

func hasNamedJSONTag(field reflect.StructField) bool {
	parts := strings.Split(field.Tag.Get("json"), ",")
	if len(parts) == 0 {
		return false
	}
	for _, s := range parts[1:] {
		if s == "inline" {
			return false
		}
	}
	return len(parts[0]) > 0
}

func (b *definitionBuilder) buildStructTypeProperty(field reflect.StructField, jsonName string, model *spec.Schema) (nameJson string, prop spec.Schema) {
	setPropertyMetadata(&prop, field)
	fieldType := field.Type
	// check for anonymous
	if len(fieldType.Name()) == 0 {
		// anonymous
		anonType := model.ID + "." + jsonName
		prop.Ref = b.createRef(fieldType, anonType)
		return jsonName, prop
	}

	if field.Name == fieldType.Name() && field.Anonymous && !hasNamedJSONTag(field) {
		// embedded struct
		sub := definitionBuilder{make(spec.Definitions), b.Config}
		sub.addModel(fieldType, "")
		subKey := sub.keyFrom(fieldType)
		// merge properties from sub
		subModel, _ := sub.Definitions[subKey]
		for k, v := range subModel.Properties {
			model.Properties[k] = v
			// if subModel says this property is required then include it
			required := false
			for _, each := range subModel.Required {
				if k == each {
					required = true
					break
				}
			}
			if required {
				model.Required = append(model.Required, k)
			}
		}
		// add all new referenced models
		for key, sub := range sub.Definitions {
			if key != subKey {
				if _, ok := b.Definitions[key]; !ok {
					b.Definitions[key] = sub
				}
			}
		}
		// empty name signals skip property
		return "", prop
	}
	// simple struct
	var pType = b.keyFrom(fieldType)
	prop.Ref = b.createRef(fieldType, pType)
	return jsonName, prop
}

func (b *definitionBuilder) buildArrayTypeProperty(field reflect.StructField, jsonName, modelName string) (nameJson string, prop spec.Schema) {
	setPropertyMetadata(&prop, field)
	fieldType := field.Type
	if fieldType.Elem().Kind() == reflect.Uint8 {
		stringt := "string"
		prop.Type = []string{stringt}
		return jsonName, prop
	}
	var pType = "array"
	prop.Type = []string{pType}
	prop.Items = &spec.SchemaOrArray{
		Schema: b.SchemaFromModel(fieldType.Elem(), modelName, jsonName),
	}
	return jsonName, prop
}

func (b *definitionBuilder) buildMapTypeProperty(field reflect.StructField, jsonName, modelName string) (nameJson string, prop spec.Schema) {
	setPropertyMetadata(&prop, field)
	fieldType := field.Type
	var pType = "object"
	prop.Type = []string{pType}

	// As long as the element isn't an interface, we should be able to figure out what the
	// intended type is and represent it in `AdditionalProperties`.
	// See: https://swagger.io/docs/specification/data-models/dictionaries/
	if fieldType.Elem().Kind().String() != "interface" {
		prop.AdditionalProperties = &spec.SchemaOrBool{
			Schema: b.SchemaFromModel(fieldType.Elem(), modelName, jsonName),
		}
	}
	return jsonName, prop
}

func (b *definitionBuilder) buildPointerTypeProperty(field reflect.StructField, jsonName, modelName string) (nameJson string, prop spec.Schema) {
	fieldType := field.Type

	prop = *b.SchemaFromModel(fieldType.Elem(), modelName, jsonName)
	setPropertyMetadata(&prop, field)
	return jsonName, prop
}

func (b *definitionBuilder) getElementTypeName(modelName, jsonName string, t reflect.Type) string {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Name() == "" {
		return modelName + "." + jsonName
	}
	return b.keyFrom(t)
}

func (b *definitionBuilder) keyFrom(st reflect.Type) string {
	key := st.String()
	if b.Config.ModelTypeNameHandler != nil {
		if name, ok := b.Config.ModelTypeNameHandler(st); ok {
			key = name
		}
	}
	if len(st.Name()) == 0 { // unnamed type
		// If it is an array, remove the leading []
		key = strings.TrimPrefix(key, "[]")
		// Swagger UI has special meaning for [
		key = strings.Replace(key, "[]", "||", -1)
	}
	return key
}

// see also https://golang.org/ref/spec#Numeric_types
func (b *definitionBuilder) isPrimitiveType(modelName string) bool {
	if len(modelName) == 0 {
		return false
	}
	return strings.Contains("uint uint8 uint16 uint32 uint64 int int8 int16 int32 int64 float32 float64 bool string byte rune time.Time", modelName)
}

// jsonNameOfField returns the name of the field as it should appear in JSON format
// An empty string indicates that this field is not part of the JSON representation
func (b *definitionBuilder) jsonNameOfField(field reflect.StructField) string {
	if jsonTag := field.Tag.Get("json"); jsonTag != "" {
		s := strings.Split(jsonTag, ",")
		if s[0] == "-" {
			// empty name signals skip property
			return ""
		} else if s[0] != "" {
			return s[0]
		}
	}
	return field.Name
}

// see also http://json-schema.org/latest/json-schema-core.html#anchor8
func (b *definitionBuilder) jsonSchemaType(modelName string) string {
	return jsonSchemaType(modelName)
}

func (b *definitionBuilder) jsonSchemaFormat(modelName string) string {
	if b.Config.SchemaFormatHandler != nil {
		if mapped := b.Config.SchemaFormatHandler(modelName); mapped != "" {
			return mapped
		}
	}
	return jsonSchemaFormat(modelName)
}
