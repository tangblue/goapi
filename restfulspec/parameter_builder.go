package restfulspec

import (
	"reflect"

	"github.com/tangblue/goapi/restful"
	"github.com/tangblue/goapi/spec"
)

type parameterBuilder struct {
	parameters map[string]*restful.Parameter
	Config     Config
}

func (b *parameterBuilder) createRef(refName string) spec.Ref {
	return spec.MustCreateRef("#/parameters/" + refName)
}

func (b *parameterBuilder) getRefParameters(defBuilder *definitionBuilder) spec.RefParameters {
	parameters := spec.RefParameters{}

	for _, v := range b.parameters {
		parameters[v.Name] = b.createParameter(v, defBuilder)
	}
	return parameters
}

func (b *parameterBuilder) build(param *restful.Parameter, pattern string, defBuilder *definitionBuilder) spec.Parameter {
	if param.RefName != "" {
		if b.parameters == nil {
			b.parameters = make(map[string]*restful.Parameter)
		}
		refName := param.RefName
		if v, ok := b.parameters[refName]; ok {
			if param != v {
				panic("parameter confilcts.")
			}
		} else {
			b.parameters[refName] = param
		}
		return spec.Parameter{Refable: spec.Refable{Ref: b.createRef(refName)}}
	}

	return b.createParameter(param, defBuilder)
}

func (b *parameterBuilder) createParameter(param *restful.Parameter, defBuilder *definitionBuilder) spec.Parameter {
	if param.Model == nil {
		return param.Parameter
	}

	if param.Required {
		param.Example = param.Model
	} else {
		param.Default = param.Model
	}

	if param.TypeName() == "" {
		typeName := reflect.TypeOf(param.Model).Kind().String()
		if !isPrimitiveType(typeName) {
			panic("parameter type is not primitive.")
		}
		if param.CollectionFormat != "" {
			param.Type = "array"
			param.Items = spec.NewItems()
			param.Items.Typed(jsonSchemaType(typeName), jsonSchemaFormat(typeName))
		} else {
			param.Typed(jsonSchemaType(typeName), jsonSchemaFormat(typeName))
		}
	}

	if param.In == "body" && param.Schema == nil {
		st := reflect.TypeOf(param.Model)
		param.Schema = defBuilder.SchemaFromModel(st, "", "")
	}

	return param.Parameter
}
