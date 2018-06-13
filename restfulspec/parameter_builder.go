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

func (b *parameterBuilder) getRefParameters() spec.RefParameters {
	parameters := spec.RefParameters{}

	for _, v := range b.parameters {
		p := v.Data()
		parameters[p.Name] = b.createParameter(v)
	}
	return parameters
}

func (b *parameterBuilder) build(r restful.Route, restfulParam *restful.Parameter, pattern string) spec.Parameter {
	param := restfulParam.Data()
	if param.RefName != "" {
		if b.parameters == nil {
			b.parameters = make(map[string]*restful.Parameter)
		}
		refName := param.RefName
		if v, ok := b.parameters[refName]; ok {
			if restfulParam != v {
				panic("parameter confilcts.")
			}
		} else {
			b.parameters[refName] = restfulParam
		}
		return spec.Parameter{Refable: spec.Refable{Ref: b.createRef(refName)}}
	}

	return b.createParameter(restfulParam)
}

func (b *parameterBuilder) createParameter(restfulParam *restful.Parameter) spec.Parameter {
	p := spec.Parameter{}
	param := restfulParam.Data()
	typeName := restfulParam.GetDataTypeName()
	p.In = asParamType(param.Kind)
	if param.AllowMultiple {
		p.Type = "array"
		p.Items = spec.NewItems()
		p.Items.Type = typeName
		p.CollectionFormat = param.CollectionFormat
	} else {
		p.Type = typeName
	}
	p.Description = param.Description
	p.Name = param.Name
	p.Required = param.Required

	if param.Kind == restful.BodyParameterKind {
		st := reflect.TypeOf(param.DefaultValue)
		p.Schema = new(spec.Schema)
		p.SimpleSchema = spec.SimpleSchema{}
		defBuilder := definitionBuilder{}
		if st.Kind() == reflect.Array || st.Kind() == reflect.Slice {
			dataTypeName := defBuilder.keyFrom(st.Elem())
			p.Schema.Type = []string{"array"}
			p.Schema.Items = &spec.SchemaOrArray{
				Schema: &spec.Schema{},
			}
			isPrimitive := isPrimitiveType(dataTypeName)
			if isPrimitive {
				mapped := jsonSchemaType(dataTypeName)
				p.Schema.Items.Schema.Type = []string{mapped}
			} else {
				p.Schema.Items.Schema.Ref = defBuilder.createRef(dataTypeName)
			}
		} else {
			p.Schema.Ref = defBuilder.createRef(typeName)
		}

	} else {
		p.Default = param.DefaultValue
		typeKind := reflect.TypeOf(param.DefaultValue).Kind().String()
		if isPrimitiveType(typeKind) {
			p.Type = jsonSchemaType(typeKind)
			p.Format = jsonSchemaFormat(typeKind)
		} else {
			p.Type = jsonSchemaType(typeName)
			p.Format = jsonSchemaFormat(typeName)
		}
		if param.DataFormat != "" {
			p.Format = param.DataFormat
		}
		if param.MinValue != nil {
			p.WithMinimum(param.MinValue, false)
		}
		if param.MaxValue != nil {
			p.WithMaximum(param.MaxValue, false)
		}
		if param.Enum != nil {
			p.WithEnum(param.Enum...)
		}
		if param.MinLength != 0 || param.MaxLength != 0 {
			p.WithMinLength(param.MinLength)
			p.WithMaxLength(param.MaxLength)
		}
		if param.Regex != "" {
			p.WithPattern(param.Regex)
		}
	}

	return p
}
