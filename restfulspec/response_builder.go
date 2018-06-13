package restfulspec

import (
	"reflect"

	"github.com/tangblue/goapi/restful"
	"github.com/tangblue/goapi/spec"
)

type responseBuilder struct {
	responses map[string]*restful.ResponseError
	Config    Config
}

func (b *responseBuilder) createRef(refName string) spec.Ref {
	return spec.MustCreateRef("#/responses/" + refName)
}

func (b *responseBuilder) getRefResponses() spec.RefResponses {
	responses := spec.RefResponses{}

	for _, e := range b.responses {
		responses[e.RefName] = b.createResponse(e)
	}
	return responses
}

func (b *responseBuilder) build(e *restful.ResponseError) spec.Response {
	if e.RefName != "" {
		if b.responses == nil {
			b.responses = make(map[string]*restful.ResponseError)
		}
		if v, ok := b.responses[e.RefName]; ok {
			if e != v {
				panic("response conflict")
			}
		} else {
			b.responses[e.RefName] = e
		}

		return spec.Response{Refable: spec.Refable{Ref: b.createRef(e.RefName)}}
	}

	return b.createResponse(e)
}

func (b *responseBuilder) createResponse(e *restful.ResponseError) (r spec.Response) {
	r.Description = e.Message
	if e.Model != nil {
		st := reflect.TypeOf(e.Model)
		if st.Kind() == reflect.Ptr {
			// For pointer type, use element type as the key; otherwise we'll
			// endup with '#/definitions/*Type' which violates openapi spec.
			st = st.Elem()
		}
		r.Schema = new(spec.Schema)
		defBuilder := definitionBuilder{}
		if st.Kind() == reflect.Array || st.Kind() == reflect.Slice {
			modelName := defBuilder.keyFrom(st.Elem())
			r.Schema.Type = []string{"array"}
			r.Schema.Items = &spec.SchemaOrArray{
				Schema: &spec.Schema{},
			}
			isPrimitive := isPrimitiveType(modelName)
			if isPrimitive {
				mapped := jsonSchemaType(modelName)
				r.Schema.Items.Schema.Type = []string{mapped}
			} else {
				r.Schema.Items.Schema.Ref = defBuilder.createRef(modelName)
			}
		} else {
			modelName := st.Kind().String()
			if !isPrimitiveType(modelName) {
				modelName = defBuilder.keyFrom(st)
			}
			if isPrimitiveType(modelName) {
				// If the response is a primitive type, then don't reference any definitions.
				// Instead, set the schema's "type" to the model name.
				r.Schema.AddType(modelName, "")
			} else {
				modelName := defBuilder.keyFrom(st)
				r.Schema.Ref = defBuilder.createRef(modelName)
			}
		}
	}
	return r
}
