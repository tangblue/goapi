package restfulspec

import (
	"reflect"
	"strings"

	"github.com/tangblue/goapi/spec"
)

func setDescription(prop *spec.Schema, field reflect.StructField) {
	if tag := field.Tag.Get("description"); tag != "" {
		prop.Description = tag
	}
}

func setDefaultValue(prop *spec.Schema, field reflect.StructField) {
	if tag := field.Tag.Get("default"); tag != "" {
		prop.Default = stringReflectType(field.Type, tag)
	}
}

func setEnumValues(prop *spec.Schema, field reflect.StructField) {
	// We use | to separate the enum values.  This value is chosen
	// since its unlikely to be useful in actual enumeration values.
	if tag := field.Tag.Get("enum"); tag != "" {
		enums := []interface{}{}
		for _, s := range strings.Split(tag, "|") {
			enums = append(enums, s)
		}
		prop.Enum = enums
	}
}

func setMaximum(prop *spec.Schema, field reflect.StructField) {
	if tag := field.Tag.Get("maximum"); tag != "" {
		prop.Maximum = stringReflectType(field.Type, tag)
	}
}

func setMinimum(prop *spec.Schema, field reflect.StructField) {
	if tag := field.Tag.Get("minimum"); tag != "" {
		prop.Minimum = stringReflectType(field.Type, tag)
	}
}

func setType(prop *spec.Schema, field reflect.StructField) {
	if tag := field.Tag.Get("type"); tag != "" {
		// Check if the first two characters of the type tag are
		// intended to emulate slice/array behaviour.
		//
		// If type is intended to be a slice/array then add the
		// overriden type to the array item instead of the main property
		if len(tag) > 2 && tag[0:2] == "[]" {
			pType := "array"
			prop.Type = []string{pType}
			prop.Items = &spec.SchemaOrArray{
				Schema: &spec.Schema{},
			}
			iType := tag[2:]
			prop.Items.Schema.Type = []string{iType}
			return
		}

		prop.Type = []string{tag}
	}
}

func setUniqueItems(prop *spec.Schema, field reflect.StructField) {
	tag := field.Tag.Get("unique")
	switch tag {
	case "true":
		prop.UniqueItems = true
	case "false":
		prop.UniqueItems = false
	}
}

func setReadOnly(prop *spec.Schema, field reflect.StructField) {
	tag := field.Tag.Get("readOnly")
	switch tag {
	case "true":
		prop.ReadOnly = true
	case "false":
		prop.ReadOnly = false
	}
}

func setPropertyMetadata(prop *spec.Schema, field reflect.StructField) {
	setDescription(prop, field)
	setDefaultValue(prop, field)
	setEnumValues(prop, field)
	setMinimum(prop, field)
	setMaximum(prop, field)
	setUniqueItems(prop, field)
	setType(prop, field)
	setReadOnly(prop, field)
}
