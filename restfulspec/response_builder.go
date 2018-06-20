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

func (b *responseBuilder) getRefResponses(defBuilder *definitionBuilder) spec.RefResponses {
	responses := spec.RefResponses{}

	for _, e := range b.responses {
		responses[e.RefName] = b.createResponse(e, defBuilder)
	}
	return responses
}

func (b *responseBuilder) build(e *restful.ResponseError, defBuilder *definitionBuilder) spec.Response {
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

	return b.createResponse(e, defBuilder)
}

func (b *responseBuilder) createResponse(e *restful.ResponseError, defBuilder *definitionBuilder) (r spec.Response) {
	if e.Schema == nil && e.Model != nil {
		st := reflect.TypeOf(e.Model)
		e.Schema = defBuilder.SchemaFromModel(st, "", "")
		for k, v := range e.Headers {
			if v.TypeName() == "" && v.Example != nil {
				name := reflect.TypeOf(v.Example).Kind().String()
				if !isPrimitiveType(name) {
					panic("Header is not primitive type")
				}
				v.Typed(jsonSchemaType(name), jsonSchemaFormat(name))
			}
			e.AddHeader(k, &v)
		}
	}
	return e.Response
}
